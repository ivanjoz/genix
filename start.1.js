#!/usr/bin/env node
// Select the alternate local environment while reusing the canonical launcher.
const path = require("path")

process.env.GENIX_CREDENTIALS_FILE = path.join(__dirname, "credentials.1.json")
// Keep this environment's public frontend entry point separate from the default launcher.
process.env.GENIX_PROXY_PORT = "3573"
console.log(`Credenciales seleccionadas: ${process.env.GENIX_CREDENTIALS_FILE}`)
console.log(`Puerto frontend seleccionado: ${process.env.GENIX_PROXY_PORT}`)

require("./start.js")
