import { spawn, type ChildProcess } from "node:child_process";
import { resolve } from "node:path";

export class GoServerManager {
  private goProcess: ChildProcess | null = null;
  constructor(
    private readonly serverEntry: string,
    private readonly goRoot: string,
  ) {}

  start(port: number): Promise<void> {
    return new Promise((resolveStart, rejectStart) => {
      const proc = spawn(
        "go",
        ["run", "-tags", "shiftapidev", this.serverEntry],
        {
          cwd: resolve(this.goRoot),
          stdio: ["ignore", "inherit", "inherit"],
          detached: true,
          env: {
            ...process.env,
            SHIFTAPI_PORT: String(port),
          },
        },
      );
      this.goProcess = proc;

      let settled = false;

      proc.on("error", (err) => {
        console.error("[shiftapi] Failed to start Go server:", err.message);
        if (!settled) {
          settled = true;
          rejectStart(err);
        }
      });

      proc.on("exit", (code) => {
        if (code !== null && code !== 0) {
          console.error(`[shiftapi] Go server exited with code ${code}`);
        }
        this.goProcess = null;
      });

      proc.on("spawn", () => {
        if (!settled) {
          settled = true;
          resolveStart();
        }
      });

      console.log(
        `[shiftapi] Go server starting on port ${port}: go run ${this.serverEntry}`,
      );
    });
  }

  stop(): Promise<void> {
    const proc = this.goProcess;
    if (!proc || !proc.pid) return Promise.resolve();

    const pid = proc.pid;
    this.goProcess = null;

    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        try {
          process.kill(-pid, "SIGKILL");
        } catch {
          // already gone
        }
        resolve();
      }, 5000);

      proc.on("exit", () => {
        clearTimeout(timeout);
        resolve();
      });

      try {
        process.kill(-pid, "SIGTERM");
      } catch {
        clearTimeout(timeout);
        resolve();
      }
    });
  }

  forceKill(): void {
    if (this.goProcess?.pid) {
      try {
        process.kill(-this.goProcess.pid, "SIGTERM");
      } catch {
        // already gone
      }
    }
  }
}
