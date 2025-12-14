#!/usr/bin/env bun
/**
 * Post-process OpenAPI spec to add oneOf for union types.
 * 
 * This script patches the converted OpenAPI 3.0 spec to add proper oneOf
 * discriminated unions where swag's Swagger 2.0 output couldn't express them.
 */

const inputFile = Bun.argv[2];
const outputFile = Bun.argv[3] || inputFile;

if (!inputFile) {
  console.error('Usage: bun patch-openapi.ts <input.json> [output.json]');
  process.exit(1);
}

const file = Bun.file(inputFile);
const spec = await file.json();

// Find the moveItemsToDirectoryResponse schema and update items to use oneOf
const schemas = spec.components?.schemas || {};

if (schemas['internal_catalog.moveItemsToDirectoryResponse']) {
  const response = schemas['internal_catalog.moveItemsToDirectoryResponse'];
  if (response.properties?.items) {
    response.properties.items = {
      type: 'array',
      description: 'Array of items included in the request (FileInfo or DirectoryInfo objects)',
      items: {
        oneOf: [
          { $ref: '#/components/schemas/internal_catalog.FileInfo' },
          { $ref: '#/components/schemas/internal_catalog.DirectoryInfo' }
        ],
        discriminator: {
          propertyName: 'kind',
          mapping: {
            file: '#/components/schemas/internal_catalog.FileInfo',
            directory: '#/components/schemas/internal_catalog.DirectoryInfo'
          }
        }
      }
    };
    console.log('✓ Patched moveItemsToDirectoryResponse.items with oneOf');
  }
}

await Bun.write(outputFile, JSON.stringify(spec, null, 2));
console.log(`✓ Written to ${outputFile}`);

