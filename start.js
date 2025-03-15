#!/usr/bin/env node
//********************************************************* */
const FILENAME = "start.js"
const FRONTEND_SCRIPT = "npm start"
const BACKEND_GO_SCRIPT = "go run -v . dev"
const ENVIROMENT_VARIABLES = ""
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

// Revisa si todo está instalado
if (!fs.existsSync("node_modules")){ 
  console.log("Instalando dependiencias de Node.js...")
  execSync('npm install', { encoding: 'utf-8' })
}

const nodemon = require("nodemon")

const frontendPath = path.join(__dirname, 'frontend')
if (!fs.existsSync(path.join(frontendPath, "node_modules"))){ 
  console.log("No se encontraron los node_modules en el frontend. Instalando...")
  if(isWindows){
    execSync(`cd ${frontendPath} & npm install`, { stdio: "inherit", shell: true })
  } else {
    execSync('cd frontend && npm install', { encoding: 'utf-8' })
  }
}

const backendGoPath = path.join(__dirname, 'backend')
console.log("Instalando los paquetes de Go (si no lo están)...")
if(isWindows){
  execSync(`cd ${backendGoPath} & go mod tidy`, { stdio: "inherit", shell: true })
} else {
  execSync('cd backend && go mod tidy', { encoding: 'utf-8' })
}

let frontendScript = `cd ./frontend && ${FRONTEND_SCRIPT}`
if (isWindows) {
  frontendScript = `cd ${frontendPath} & ${FRONTEND_SCRIPT}`
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
// const MAGENTA_BAR = "\x1b[45m \x1b[0m"

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
    cwd: "backend",    // Sets the working directory to backend-golang
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
  console.log(`${YELLOW_BAR}${YELLOW_BAR} Frontend   ${BLUE_BAR}${BLUE_BAR} Backend (Go)`)

  // Run all scripts in parallel
  runScript(frontendScript, YELLOW_BAR)
  startBackendGo()
}

// Function to run a script and capture output
const runScript = (script, name) => {
  console.log("Ejecutando script:", script)

  const scriptProcess = isWindows ?
    spawn(script, [], { 
      stdio: ["ignore", "pipe", "pipe"], detached: false, shell: true }) 
    :
    spawn("bash", ["-c", script], {
      stdio: ["ignore", "pipe", "pipe"], // Capture stdout and stderr
      detached: false, // Ensures process terminates when Node.js exits
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

// Kill necessary ports before starting
[3588, 3589].forEach(killPortIfInUse)
// Run scripts
runScripts()
