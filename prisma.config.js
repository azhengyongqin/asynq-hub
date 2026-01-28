import { defineConfig, env } from "prisma/config";
import { config } from "dotenv";

// 加载 .env 文件
config();

export default defineConfig({
  schema: "prisma/schema.prisma",
  migrations: {
    path: "prisma/migrations",
  },
  datasource: {
    url: env("DATABASE_URL"),
  },
});
