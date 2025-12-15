package catalog

import (
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	vfs *virtualfs.VirtualFS
	db  *bun.DB
}

// patchFileRequest represents a file update request
// @Description Request to update file properties
type patchFileRequest struct {
	// New name for the file
	Name string `json:"name" example:"renamed-document.pdf"`
}

// patchDirectoryRequest represents a directory update request
// @Description Request to update directory properties
type patchDirectoryRequest struct {
	// New name for the directory
	Name string `json:"name" example:"My Documents"`
}

func NewHTTPHandler(vfs *virtualfs.VirtualFS, db *bun.DB) *HTTPHandler {
	return &HTTPHandler{vfs: vfs, db: db}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	api.Delete("/files", h.deleteFiles)

	fg := api.Group("/files/:fileID")
	fg.Use(h.currentFileMiddleware)
	fg.Get("/", h.fetchFile)
	fg.Get("/content", h.downloadFile)
	fg.Patch("/", h.patchFile)
	fg.Delete("/", h.deleteFile)

	api.Post("/directories", h.createDirectory)
	api.Delete("/directories", h.deleteDirectories)

	dg := api.Group("/directories/:directoryID")
	dg.Use(h.currentDirectoryMiddleware)
	dg.Get("/", h.fetchDirectory)
	dg.Post("/content", h.moveItemsToDirectory)
	dg.Get("/content", h.listDirectory)
	dg.Patch("/", h.patchDirectory)
	dg.Delete("/", h.deleteDirectory)
}

func fileInfoFromNode(node *virtualfs.Node) FileInfo {
	return FileInfo{
		Kind:     DirItemKindFile,
		ID:       node.PublicID,
		Name:     node.Name,
		Size:     node.Size,
		MimeType: node.MimeType,
	}
}

func directoryInfoFromNode(node *virtualfs.Node) DirectoryInfo {
	return DirectoryInfo{
		Kind:      DirItemKindDirectory,
		ID:        node.PublicID,
		Name:      node.Name,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
		DeletedAt: node.DeletedAt,
	}
}

func toDirectoryItem(node *virtualfs.Node) any {
	switch node.Kind {
	default:
		return FileInfo{}
	case virtualfs.NodeKindDirectory:
		return directoryInfoFromNode(node)
	case virtualfs.NodeKindFile:
		return fileInfoFromNode(node)
	}
}
