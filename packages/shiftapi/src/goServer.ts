import { spawn, type ChildProcess } from "node:child_process";
import { resolve } from "node:path";

export class GoServerManager {
  private goProcess: ChildProcess | null = null;
  private _devPort: number | null = null;

  constructor(
    private readonly serverEntry: string,
    private readonly goRoot: string,
  ) {}

  /** The dev port parsed from the Go server's stderr, or null if not yet known. */
  get devPort(): number | null {
    return this._devPort;
  }

  start(): Promise<void> {
    this._devPort = null;

    return new Promise((resolveStart, rejectStart) => {
      const proc = spawn(
        "go",
        ["run", "-tags", "shiftapidev", this.serverEntry],
        {
          cwd: resolve(this.goRoot),
          stdio: ["ignore", "inherit", "pipe"],
          detached: true,
          env: {
            ...process.env,
          },
        },
      );
      this.goProcess = proc;

      let settled = false;
      let stderrBuf = "";

      proc.stderr?.on("data", (chunk: Buffer) => {
        const text = chunk.toString();
        // Forward stderr to the console so the user sees Go logs.
        process.stderr.write(text);

        // Parse the dev port line emitted by devInit.
        if (this._devPort === null) {
          stderrBuf += text;
          const match = stderrBuf.match(/shiftapi:dev:port=(\d+)/);
          if (match) {
            this._devPort = parseInt(match[1], 10);
          }
        }
      });

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
        `[shiftapi] Go server starting: go run -tags shiftapidev ${this.serverEntry}`,
      );
    });
  }

  /**
   * Waits until the dev port has been parsed from stderr.
   * Returns the port number.
   */
  async waitForDevPort(timeout = 30_000): Promise<number> {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      if (this._devPort !== null) return this._devPort;
      if (this.goProcess === null) {
        throw new Error("Go server exited before reporting dev port");
      }
      await new Promise((resolve) => setTimeout(resolve, 50));
    }
    throw new Error(
      `Timed out waiting for dev port from Go server after ${timeout}ms`,
    );
  }

  stop(): Promise<void> {
    const proc = this.goProcess;
    if (!proc || !proc.pid) return Promise.resolve();

    const pid = proc.pid;
    this.goProcess = null;
    this._devPort = null;

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
