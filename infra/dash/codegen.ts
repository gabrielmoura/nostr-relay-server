import type { CodegenConfig } from "@graphql-codegen/cli"

const config: CodegenConfig = {
  schema: ["../../graph/schema_*.graphqls"],
  documents: ["src/graphql/**/*.{ts,tsx}"],
  ignoreNoDocuments: false,
  generates: {
    "src/graphql/generated/operations.ts": {
      plugins: ["typescript", "typescript-operations", "typed-document-node"],
      config: {
        useTypeImports: true,
        skipTypename: false,
        nonOptionalTypename: true,
        enumsAsTypes: true,
        scalars: {
          Int64: "number",
          JSON: "Record<string, unknown>",
          Upload: "File",
        },
      },
    },
  },
}

export default config
