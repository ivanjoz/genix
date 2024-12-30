import path from 'path'
import fs from 'fs'
const __dirname = new URL('.', import.meta.url).pathname;

const buildFolder = path.join(__dirname,'.output','public')
const deployFolder = path.join(__dirname,'..','docs')

const buildSubFolder = [
  "", "images", "pdf", "assets", "libs", "_build", path.join("_build","assets")
]

console.log("Moviendo al folder:", deployFolder)

for(const folder of buildSubFolder){
  // console.log("Copiando folder:",folder)
  const buildPath = folder ? path.join(buildFolder,folder) : buildFolder
  const deployPath = folder ? path.join(deployFolder,folder) : deployFolder

  if(!fs.existsSync(buildPath)){
    continue
  }

  console.log("Copiando al folder destino: ", deployPath)
  if(folder){
    if(fs.existsSync(deployPath)){
      fs.rmSync(deployPath, { recursive: true, force: true })
    }
    fs.mkdirSync(deployPath)
  }
  
  const files = fs.readdirSync(buildPath)

  for(const file of files){
    const filePath = path.join(buildPath,file)
    if(fs.lstatSync(filePath).isDirectory()){ continue }
    
    const fileArray = file.split(".")
    const ext = fileArray[fileArray.length - 1]
    if(ext === "gz" || ext === "br"){ continue }
    console.log("Moviendo: ", path.join(deployPath,file))
    fs.copyFileSync(filePath, path.join(deployPath, file))
  }
}

console.log("Moviendo 404.html...")
fs.copyFileSync(path.join(buildFolder,"index.html"), path.join(deployFolder,"404.html"))
