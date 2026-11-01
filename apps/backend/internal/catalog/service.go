package catalog

import (
	"context"
	"errors"

	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrParentNotFound     = errors.New("parent not found")
	ErrParentNotDirectory = errors.New("parent is not a directory")
	ErrItemsNotFound      = errors.New("one or more items not found")
)

type DirectoryResult struct {
	Node *virtualfs.Node
	Path virtualfs.Path
}

type Service interface {
	FindNode(ctx context.Context, publicID string, scope *virtualfs.Scope) (*virtualfs.Node, error)
	FindDirectory(ctx context.Context, id string, scope *virtualfs.Scope) (*virtualfs.Node, error)
	ReadFile(ctx context.Context, node *virtualfs.Node, scope *virtualfs.Scope) (virtualfs.FileContent, error)
	RealPath(ctx context.Context, node *virtualfs.Node, scope *virtualfs.Scope) (virtualfs.Path, error)
	ListChildren(ctx context.Context, node *virtualfs.Node, opts virtualfs.ListChildrenOptions, scope *virtualfs.Scope) ([]*virtualfs.Node, *virtualfs.ListChildrenCursor, error)
	CreateDirectory(ctx context.Context, parentID, name string, includePath bool, scope *virtualfs.Scope) (*DirectoryResult, error)
	RenameNode(ctx context.Context, node *virtualfs.Node, name string, scope *virtualfs.Scope) error
	DeleteNode(ctx context.Context, node *virtualfs.Node, trash bool, scope *virtualfs.Scope) (*virtualfs.Node, error)
	DeleteNodes(ctx context.Context, ids []string, kind virtualfs.NodeKind, trash bool, scope *virtualfs.Scope) ([]*virtualfs.Node, error)
	MoveNodes(ctx context.Context, ids []string, targetID uuid.UUID, scope *virtualfs.Scope) (*virtualfs.MoveFilesResult, error)
}

type service struct {
	db  *bun.DB
	vfs *virtualfs.VirtualFS
}

var _ Service = (*service)(nil)

func NewService(db *bun.DB, vfs *virtualfs.VirtualFS) Service {
	return &service{db: db, vfs: vfs}
}

func (s *service) FindNode(ctx context.Context, publicID string, scope *virtualfs.Scope) (*virtualfs.Node, error) {
	return s.vfs.FindNodeByPublicID(ctx, s.db, publicID, scope)
}

func (s *service) FindDirectory(ctx context.Context, id string, scope *virtualfs.Scope) (*virtualfs.Node, error) {
	if scope == nil {
		return nil, virtualfs.ErrAccessDenied
	}

	if id == "root" {
		return s.vfs.FindNode(ctx, s.db, scope.RootNodeID.String(), scope)
	}

	return s.FindNode(ctx, id, scope)
}

func (s *service) ReadFile(ctx context.Context, node *virtualfs.Node, scope *virtualfs.Scope) (virtualfs.FileContent, error) {
	return s.vfs.ReadFile(ctx, s.db, node, scope)
}

func (s *service) RealPath(ctx context.Context, node *virtualfs.Node, scope *virtualfs.Scope) (virtualfs.Path, error) {
	return s.vfs.RealPath(ctx, s.db, node, scope)
}

func (s *service) ListChildren(ctx context.Context, node *virtualfs.Node, opts virtualfs.ListChildrenOptions, scope *virtualfs.Scope) ([]*virtualfs.Node, *virtualfs.ListChildrenCursor, error) {
	return s.vfs.ListChildren(ctx, s.db, node, opts, scope)
}

func (s *service) CreateDirectory(ctx context.Context, parentID, name string, includePath bool, scope *virtualfs.Scope) (*DirectoryResult, error) {
	var result DirectoryResult

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		parent, err := s.vfs.FindNodeByPublicID(ctx, tx, parentID, scope)
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return ErrParentNotFound
		}

		if err != nil {
			return err
		}

		if parent.Kind != virtualfs.NodeKindDirectory {
			return ErrParentNotDirectory
		}

		result.Node, err = s.vfs.CreateDirectory(ctx, tx, parent.ID, name, scope)
		if err != nil {
			return err
		}

		if includePath {
			result.Path, err = s.vfs.RealPath(ctx, tx, result.Node, scope)
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *service) RenameNode(ctx context.Context, node *virtualfs.Node, name string, scope *virtualfs.Scope) error {
	if name == "" {
		return nil
	}

	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.vfs.RenameNode(ctx, tx, node, name, scope)
	})
}

func (s *service) DeleteNode(ctx context.Context, node *virtualfs.Node, trash bool, scope *virtualfs.Scope) (*virtualfs.Node, error) {
	var result *virtualfs.Node

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if !trash {
			return s.vfs.PermanentlyDeleteNode(ctx, tx, node, scope)
		}

		var err error

		result, err = s.vfs.SoftDeleteNode(ctx, tx, node, scope)

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *service) DeleteNodes(ctx context.Context, ids []string, kind virtualfs.NodeKind, trash bool, scope *virtualfs.Scope) ([]*virtualfs.Node, error) {
	var result []*virtualfs.Node

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		nodes, err := s.vfs.FindNodesByPublicID(ctx, tx, ids, scope)
		if err != nil {
			return err
		}

		if len(nodes) == 0 {
			return nil
		}

		for _, node := range nodes {
			if node.Kind != kind {
				return virtualfs.ErrUnsupportedOperation
			}
		}

		if trash {
			result, err = s.vfs.SoftDeleteNodes(ctx, tx, nodes, scope)

			return err
		}

		if kind == virtualfs.NodeKindFile {
			return s.vfs.PermanentlyDeleteFiles(ctx, tx, nodes, scope)
		}

		for _, node := range nodes {
			if err := s.vfs.PermanentlyDeleteNode(ctx, tx, node, scope); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *service) MoveNodes(ctx context.Context, ids []string, targetID uuid.UUID, scope *virtualfs.Scope) (*virtualfs.MoveFilesResult, error) {
	var result *virtualfs.MoveFilesResult

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		nodes, err := s.vfs.FindNodesByPublicID(ctx, tx, ids, scope)
		if err != nil {
			return err
		}

		if len(nodes) != len(ids) {
			return ErrItemsNotFound
		}

		result, err = s.vfs.MoveNodesInSameDirectory(ctx, tx, nodes, targetID, scope)

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
