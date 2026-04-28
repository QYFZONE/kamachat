import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const httpTarget = env.VITE_PROXY_TARGET || "http://127.0.0.1:8080";
  const wsTarget = env.VITE_PROXY_WS_TARGET || "ws://127.0.0.1:8080";

  return {
    plugins: [vue()],
    server: {
      port: 5173,
      host: "0.0.0.0",
      proxy: {
        "/auth": httpTarget,
        "/sms": httpTarget,
        "/user": httpTarget,
        "/group": httpTarget,
        "/contact": httpTarget,
        "/session": httpTarget,
        "/message": httpTarget,
        "/wss": {
          target: wsTarget,
          ws: true,
        },
      },
    },
  };
});
