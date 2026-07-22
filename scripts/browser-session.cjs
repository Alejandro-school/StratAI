const { chromium } = require("playwright");
const readline = require("node:readline");
const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const profileDir = path.join(root, ".browser-profile");
const defaultUrl = process.env.BROWSER_URL || "http://localhost:3000/";

function arg(name, fallback) {
  const index = process.argv.indexOf(name);
  return index >= 0 && process.argv[index + 1] ? process.argv[index + 1] : fallback;
}

function waitForEnter(message) {
  const input = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolve) => input.question(`${message}\n`, () => {
    input.close();
    resolve();
  }));
}

async function openContext(headless) {
  fs.mkdirSync(profileDir, { recursive: true });
  return chromium.launchPersistentContext(profileDir, {
    headless,
    viewport: { width: 1440, height: 1000 },
  });
}

async function login() {
  const context = await openContext(false);
  const page = context.pages()[0] || await context.newPage();
  await page.goto(arg("--url", defaultUrl), { waitUntil: "domcontentloaded" });
  await waitForEnter("Completa el inicio de sesión de Steam en la ventana del navegador y pulsa Enter aquí cuando hayas terminado.");
  console.log(`Sesión guardada localmente. URL actual: ${page.url()}`);
  await context.close();
}

async function inspect() {
  const context = await openContext(true);
  const page = context.pages()[0] || await context.newPage();
  await page.goto(arg("--url", defaultUrl), { waitUntil: "networkidle" });
  console.log(JSON.stringify({
    title: await page.title(),
    url: page.url(),
    text: (await page.locator("body").innerText()).slice(0, 20000),
  }, null, 2));
  const screenshot = arg("--screenshot", null);
  if (screenshot) await page.screenshot({ path: path.resolve(root, screenshot), fullPage: true });
  await context.close();
}

const command = process.argv[2];
if (!["login", "inspect"].includes(command)) {
  console.error("Uso: node scripts/browser-session.cjs <login|inspect> [--url URL] [--screenshot ARCHIVO]");
  process.exit(1);
}

(command === "login" ? login : inspect)().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
