/// <reference types="vite/client" />

interface Window {
  COMPASSO_CONFIG?: { apiBaseUrl: string };
}

declare module "*.module.css" {
  const classes: Record<string, string>;
  export default classes;
}
