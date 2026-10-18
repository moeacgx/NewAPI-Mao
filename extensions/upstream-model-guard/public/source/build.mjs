import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const directory = path.dirname(fileURLToPath(import.meta.url));
const translations = JSON.parse(await fs.readFile(path.join(directory, "locales.json"), "utf8"));
const shared = (await fs.readFile(path.join(directory, "page.mjs"), "utf8")).replace(
  "export function createGuardPage",
  "function createGuardPage",
);
await fs.mkdir(path.join(directory, "../native"), { recursive: true });
for (const target of ["default", "classic"]) {
  const adapter = await fs.readFile(path.join(directory, `${target}.mjs`), "utf8");
  await fs.writeFile(
    path.join(directory, `../native/${target}.mjs`),
    `// oxfmt-ignore\nconst guardTranslations = ${JSON.stringify(translations)};\n${shared}\n${adapter}`,
    "utf8",
  );
}
