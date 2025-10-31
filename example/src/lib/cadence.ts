import { readFileSync } from "fs";
import { join } from "path";

export function loadCadence(path: string): string {
  const fullPath = join(process.cwd(), path);
  return readFileSync(fullPath, "utf-8");
}
