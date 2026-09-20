-- Start Neovim inside `nix develop`; enable vim.o.exrc in your main config.
local function disable_lsp_formatting(client)
	client.server_capabilities.documentFormattingProvider = false
	client.server_capabilities.documentRangeFormattingProvider = false
end

vim.lsp.config("ts_ls", {
	cmd = { "typescript-language-server", "--stdio" },
	filetypes = { "javascript", "javascriptreact", "typescript", "typescriptreact" },
	root_markers = { "tsconfig.json", "jsconfig.json", "package.json", ".git" },
	on_attach = disable_lsp_formatting,
})

vim.lsp.config("tailwindcss", {
	cmd = { "tailwindcss-language-server", "--stdio" },
	filetypes = { "html", "css", "javascriptreact", "typescriptreact" },
	root_markers = { "vite.config.ts", "package.json", ".git" },
})

vim.lsp.config("biome", {
	cmd = { "biome", "lsp-proxy" },
	filetypes = { "javascript", "javascriptreact", "typescript", "typescriptreact", "json", "jsonc", "css" },
	root_markers = { "biome.json", "biome.jsonc" },
	on_attach = disable_lsp_formatting,
})

vim.lsp.config("gopls", {
	cmd = { "gopls" },
	filetypes = { "go", "gomod", "gowork", "gotmpl" },
	root_markers = { "go.work", "go.mod", ".git" },
})

vim.lsp.enable({ "ts_ls", "tailwindcss", "biome", "gopls" })

local ok, conform = pcall(require, "conform")
if not ok then
	vim.notify("conform.nvim not loaded", vim.log.levels.WARN)
else
	conform.setup({
		formatters_by_ft = {
			javascript = { "biome-check" },
			javascriptreact = { "biome-check" },
			typescript = { "biome-check" },
			typescriptreact = { "biome-check" },
			json = { "biome" },
			jsonc = { "biome" },
			css = { "biome" },
			go = { "goimports", "gofmt" },
			nix = { "nixfmt" },
		},
	})

	vim.keymap.set("n", "<leader>f", function()
		conform.format({ async = true, lsp_format = "fallback" })
	end, { desc = "Format buffer" })
end
