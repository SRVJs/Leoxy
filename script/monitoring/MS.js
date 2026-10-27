import { execFile } from "node:child_process"
import { appendFile, mkdir } from "node:fs/promises"
import path from "node:path"
import { promisify } from "node:util"

const execFileAsync = promisify(execFile)

const config = {
     processName: process.argv[2] ?? "leoxy",
     intervalMs: 1000,
     output: path.resolve("../../docs/baseline/ramMS.csv")
}
let stopped = false

function stop() {
     stopped = true
}

process.on("SIGINT", stop)
process.on("SIGTERM", stop)

async function getProcess() {
     const script = `$items = Get-Process -Name '${config.processName.replace(/'/g, "''")}' 
                                                  SilentlyContinue
                                                       if ($items) {
                                                            $items | Select-Object Id, ProcessName, WorkingSet64, 
                                                            PrivateMemorySize64 | ConvertTo-Json -Compress
                                                       }
     `

     const { stdout } = await execFileAsync("powershell.exe", [
          "-NoProfile",
          "-NonInteractive",
          "-Command",
          script
     ], {
          windowsHide: true,
          timeout: 5000
     })

     if (!stdout.trim()) return []

     const result = JSON.parse(stdout)

     return Array.isArray(result) ? result : [result]
}

async function main() {
     await mkdir(path.dirname(config.output), { recursive: true })

     await appendFile(
          config.output,
          "Timestamp,PID,ProcessName,WorkingSetMB,PrivateMemoryMB\n",
          "utf-8"
     )

     console.log(`Monitoring process: ${config.processName}`)
     console.log(`Output: ${config.output}`)
     console.log(`Press Ctrl+C to stop`)

     while (!stopped) {
          try {
               const processes = await getProcess()
               const timestamp = new Date().toISOString()

               for (const item of processes) {
                    const workingMB = (
                         Number(item.WorkingSet64) / 1024 ** 2
                    ).toFixed(2)
                    const privateMemoryMB = (
                         Number(item.PrivateMemorySize64) / 1024 ** 2
                    ).toFixed(2)

                    const row = [
                         timestamp,
                         item.Id,
                         item.ProcessName,
                         workingMB,
                         privateMemoryMB
                    ].join(",")

                    await appendFile(config.output, `${row}\n`, "utf-8")
                    console.log(`Time: ${timestamp}, PID:${item.Id}, RAM:${workingMB} MB, Private:${privateMemoryMB} MB`)
               }

               if (processes.length === 0) {
                    console.log(`${timestamp} Process not found ${config.processName}`)
               }
          }
          catch (err) {
               console.error("Failed to collect RAM metrics: ", err.message)
          }

          await new Promise((resolve) => setTimeout(resolve, config.intervalMs))
     }
}

main().catch(err => {
     console.error(err)
     process.exitCode = 1
})
