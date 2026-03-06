/// <reference types="vite/client" />

// ImportMetaEnv 用于声明前端运行时可读取的环境变量。
// 这里显式列出变量，是为了让 TypeScript 在访问时具备类型提示与约束。
interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string;
}

// ImportMeta 把上面的环境变量类型挂接到 `import.meta.env` 上。
interface ImportMeta {
  readonly env: ImportMetaEnv;
}
