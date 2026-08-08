#!/usr/bin/env node
//********************************************************* */
// Reopen the launcher variant that the user actually selected.
const FILENAME = require("path").basename(process.argv[1] || "start.js")
const FRONTEND_SCRIPT = "npm run dev"
// Sin -v: el listado de paquetes no aporta y ensucia el log en cada reinicio.
const BACKEND_GO_SCRIPT = "go run . dev"
const RATE_LIMITER_SCRIPT = "cargo run"
const ENVIROMENT_VARIABLES = "dev1"
const FRONTEND_PROXY_PORT = Number.parseInt(process.env.GENIX_PROXY_PORT || "3572", 10)
//********************************************************* */

const { spawn, execSync } = require("child_process")
const { platform } = require("os")
const path = require("path")
const isWindows = platform() === "win32"
const isMac = platform() === "darwin"
const fs = require("fs")

const isConsole = process.stdout.isTTY; // Check if stdout is TTY (Terminal)
// Esto solamente aplica si se inicia con doble click (sin consola)
if(!isConsole){
  if (platform() === "linux") {
    execSync(`source ~/.profile && source ~/.bashrc && konsole --hold -e "node ${FILENAME}"`)
  } else {
    console.error("Unsupported platform. Please add support for your platform.");
  }
  return
}

// Copia los skills de .agents/skills -> .claude/skills (project-level, formato <name>/SKILL.md)
const agentsSkillsPath = path.join(__dirname, '.agents', 'skills')
const claudeSkillsPath = path.join(__dirname, '.claude', 'skills')
fs.mkdirSync(agentsSkillsPath, { recursive: true })
fs.mkdirSync(claudeSkillsPath, { recursive: true })
const copyDirSync = (src, dest) => {
  fs.mkdirSync(dest, { recursive: true })
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const srcPath = path.join(src, entry.name)
    const destPath = path.join(dest, entry.name)
    if (entry.isDirectory()) copyDirSync(srcPath, destPath)
    else fs.copyFileSync(srcPath, destPath)
  }
}
for (const entry of fs.readdirSync(agentsSkillsPath, { withFileTypes: true })) {
  const src = path.join(agentsSkillsPath, entry.name)
  const dest = path.join(claudeSkillsPath, entry.name)
  if (entry.isDirectory()) copyDirSync(src, dest)
  else fs.copyFileSync(src, dest)
}
console.log(`Skills copiados: .agents/skills -> .claude/skills`)

// Revisa si todo está instalado. Todas las rutas salen de __dirname, nunca del CWD: el
// launcher se puede invocar desde cualquier carpeta (o con doble click, que abre konsole
// en el home) y tiene que encontrar el repo igual.
if (!fs.existsSync(path.join(__dirname, "node_modules"))){
  console.log("Instalando dependiencias de Node.js...")
  execSync('npm install', { encoding: 'utf-8', cwd: __dirname })
}

const nodemon = require("nodemon")

// Frontend 1
const frontendPath = path.join(__dirname, 'frontend')
const frontendNodeModules = path.join(frontendPath, "node_modules")
// OJO: revisar un paquete que realmente se instala. Antes se buscaba "rolldown-vite"
// (vite 8 ya trae rolldown adentro, ese paquete no existe aquí) por lo que la
// condición era siempre verdadera y bun install corría en cada arranque.
// package.json en vez de .bin/vite: el nombre del shim cambia según plataforma.
const viteInstalled = fs.existsSync(path.join(frontendNodeModules, "vite", "package.json"))
const storeNodeModules = fs.existsSync(path.join(frontendPath, "webpage", "node_modules"))
if (!fs.existsSync(frontendNodeModules) || !viteInstalled || !storeNodeModules){
  console.log("No se encontraron los node_modules o faltan dependencias en el frontend. Instalando...")
  execSync('bun install', { stdio: "inherit", shell: true, cwd: frontendPath })
}

// Rate limiter + SSE bridge (server_utils/, un solo binario Rust con los dos servicios)
const rateLimiterPath = path.join(__dirname, 'server_utils')

// Backend
const backendGoPath = path.join(__dirname, 'backend')

// go mod tidy reescribe go.mod/go.sum, y eso invalida el build cache de Go: la
// compilación en frío toma ~36s vs ~5s en caliente. Por eso solo corre cuando
// go.mod/go.sum cambiaron de verdad (se compara contra un stamp en tmp/).
const goModStampPath = path.join(__dirname, 'tmp', '.go-mod-stamp')
const goModFiles = ['go.mod', 'go.sum'].map(name => path.join(backendGoPath, name))
const goModStamp = goModFiles.map(file => {
  const stat = fs.existsSync(file) ? fs.statSync(file) : null
  return stat ? `${path.basename(file)}:${stat.size}:${stat.mtimeMs}` : `${path.basename(file)}:missing`
}).join('|')

const runGoModTidy = () => {
  console.log("Instalando los paquetes de Go (si no lo están)...")
  execSync('go mod tidy', { stdio: "inherit", shell: true, cwd: backendGoPath })
  // Se releen los stats: go mod tidy pudo haber reescrito los archivos.
  const nextStamp = goModFiles.map(file => {
    const stat = fs.existsSync(file) ? fs.statSync(file) : null
    return stat ? `${path.basename(file)}:${stat.size}:${stat.mtimeMs}` : `${path.basename(file)}:missing`
  }).join('|')
  fs.mkdirSync(path.dirname(goModStampPath), { recursive: true })
  fs.writeFileSync(goModStampPath, nextStamp)
}

const previousGoModStamp = fs.existsSync(goModStampPath)
  ? fs.readFileSync(goModStampPath, 'utf-8').trim()
  : null

if (previousGoModStamp === goModStamp) {
  console.log("Paquetes de Go sin cambios, se omite go mod tidy.")
} else {
  runGoModTidy()
}

// Remove enviroment variables
const ENV_PATH = path.join(__dirname, '.env')
if (fs.existsSync(ENV_PATH)) { fs.unlinkSync(ENV_PATH) }

if(ENVIROMENT_VARIABLES){
  const TMP_ENV_PATH = path.join(__dirname, 'tmp', ENVIROMENT_VARIABLES)
  if (fs.existsSync(TMP_ENV_PATH)) {
    fs.copyFileSync(TMP_ENV_PATH, ENV_PATH);
    console.log(`Variables de entorno seteadas desde ./tmp/${ENVIROMENT_VARIABLES}`)
  } else {
    console.log(`No se encontraron las variables de entorno en: ./tmp/${ENVIROMENT_VARIABLES}`)
  }
}

const BLUE_BAR = "\x1b[44m \x1b[0m"
const CYAN_BAR = "\x1b[46m \x1b[0m"
const YELLOW_BAR = "\x1b[43m \x1b[0m"
const MAGENTA_BAR = "\x1b[45m \x1b[0m"

const backendLog = (data) => {
  for(const line of (data||"").split("\n").filter(x => x.trim().length > 0)){
    process.stdout.write(`${BLUE_BAR} ${line}\n`);
  }
}

const startBackendGo = () => {
  nodemon({
    exec: BACKEND_GO_SCRIPT,  // Runs the Go application
    watch: ["."],             // Only watches the backend-golang folder
    ext: "go",                // Only watches Go files
    cwd: backendGoPath,       // Absoluto: el launcher no depende del CWD
    delay: "200ms",
    spawn: true,
    signal: "SIGTERM",
    stdout: false
  })
  .on("restart", () => {
    console.log("\n♻️ Restarting Go server...");
  })
  .on("readable", function () {
    // Read Nodemon's stdout and stderr streams
    this.stdout.on("data", (data) => backendLog(String(data)))
    this.stderr.on("data", (data) => backendLog(String(data)))
  })
}

const runScripts = () => {
  /*
  console.log("Pre-building...")
  try {
    const output = execSync('node prebuild.js', { encoding: 'utf-8' })
    console.log("Script Output:", output)
  } catch (error) {
    console.error("Error executing script:", error)
  }
  */
  console.log("Executing processes...")
  console.log(`${YELLOW_BAR}${YELLOW_BAR} Frontend   ${BLUE_BAR}${BLUE_BAR} Backend (Go)   ${MAGENTA_BAR}${MAGENTA_BAR} Rate limiter (Rust)`)

  // Run all scripts in parallel
  runScript(FRONTEND_SCRIPT, YELLOW_BAR, frontendPath)
  startBackendGo()
  runScript(RATE_LIMITER_SCRIPT, MAGENTA_BAR, rateLimiterPath, {
    RUST_LOG: process.env.RUST_LOG || "genix_server_utils=debug"
  })
}

// Function to run a script and capture output.
// workingDirectory se pasa por la opción cwd de spawn en vez de anteponer un `cd`: así no hay
// que citar rutas con espacios ni duplicar el comando por plataforma.
const runScript = (script, name, workingDirectory, environment = {}) => {
  console.log("Ejecutando script:", script, "::", workingDirectory)

  const scriptProcess = isWindows ?
    spawn(script, [], {
      stdio: ["ignore", "pipe", "pipe"], detached: false, shell: true,
      cwd: workingDirectory, env: { ...process.env, ...environment } })
    :
    spawn("bash", ["-c", script], {
      stdio: ["ignore", "pipe", "pipe"], // Capture stdout and stderr
      detached: false, // Ensures process terminates when Node.js exits
      cwd: workingDirectory,
      env: { ...process.env, ...environment }
    });

  const logdata = (prefix, data) => {
    const lines = data.toString().split("\n").filter(x => x)
    for (const line of lines) {
      console.log(`${prefix} ${line}`);
    }
  }

  scriptProcess.stdout.on("data", (data) => logdata(name, data))
  scriptProcess.stderr.on("data", (data) => logdata(name, data))
  scriptProcess.on("exit", (code) => logdata(name, `Exited with code ${code}`))
}

// Function to check if a port is in use
const killPortIfInUse = (port) => {
  if(isWindows){
    try {
      const command = `netstat -ano | findstr :${port}`;
      const responseRaw = execSync(command, { encoding: "utf-8" });

      if (!responseRaw) return;

      const lines = responseRaw.trim().split("\n");
      const pids = new Set(); // Using a Set to avoid duplicates

      lines.forEach((line) => {
        const parts = line.trim().split(/\s+/);
        const pid = parts[parts.length - 1]; // The PID is the last column
        if (pid && !isNaN(pid)) {
          pids.add(pid);
        }
      });

      if (pids.size === 0) return;

      console.log(`Port ${port} is being used by PIDs: ${[...pids].join(", ")}`);

      pids.forEach((pid) => {
        try {
          execSync(`taskkill /PID ${pid} /F`);
          console.log(`Process ${pid} killed successfully.`);
        } catch (error) {
          console.error(`Failed to kill process ${pid}: ${error.message}`);
        }
      });
    } catch (error) {
      return;
    }
  } else {
    const command = `lsof -i :${port} -t`;
    try {
      const responseRaw = execSync(command)
      if (!responseRaw) { return }
      const response = responseRaw.toString().trim()
      if (response.length <= 2) { return }
      const pids = response.split("\n").map(x => x.trim()).filter(x => x)
      console.log(`Port ${port} is being used by PIDs: ${pids.join(", ")}`);
      for (const pid of pids) {
        try {
          execSync(`kill -9 ${pid}`);
          console.log(`Process ${pid} killed successfully.`);
        } catch (error) {
          console.error(`Failed to kill process ${pid}: ${error.message}`);
        }
      }
    } catch (error) {
      return
    }
  }
}

// Kill necessary ports before starting.
// Clear the selected proxy port so launcher variants do not retain stale processes.
[3570, 3571, FRONTEND_PROXY_PORT, 3588, 3589, 14013].forEach(killPortIfInUse)
// Run scripts
runScripts()
