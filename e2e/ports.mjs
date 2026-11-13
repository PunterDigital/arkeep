// Spare ports so a local run doesn't clash with a real Arkeep on 8080/9090.
export const HTTP_PORT = Number(process.env.E2E_HTTP_PORT ?? 18080);
export const GRPC_PORT = Number(process.env.E2E_GRPC_PORT ?? 19090);
