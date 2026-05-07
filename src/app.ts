import type { Widgets } from "neo-blessed";
import type { Branch, WorkflowRun, Job, AppState, Panel, LogEntry } from "./types";
import * as gh from "./gh";
import {
  createUI,
  updateBranchesList,
  updateRunsList,
  buildRunDetailsText,
  setActivePanel,
  updateStatusBar,
  enterLogView,
  exitLogView,
  setLogContent,
  type UIComponents,
} from "./ui";

const POLL_INTERVAL_MS = 10_000; // 10 seconds for real-time updates

export class App {
  private state: AppState;
  private ui: UIComponents;
  private pollTimers: { branches: ReturnType<typeof setInterval> | null; runs: ReturnType<typeof setInterval> | null } = {
    branches: null,
    runs: null,
  };

  private constructor(state: AppState, ui: UIComponents) {
    this.state = state;
    this.ui = ui;
  }

  static async create(owner?: string, repo?: string): Promise<App> {
    let resolvedOwner = owner;
    let resolvedRepo = repo;

    if (!resolvedOwner || !resolvedRepo) {
      const detected = await gh.detectOwnerRepo();
      resolvedOwner = detected.owner;
      resolvedRepo = detected.repo;
    }

    const ui = createUI();
    const state: AppState = {
      owner: resolvedOwner,
      repo: resolvedRepo,
      branches: [],
      runs: [],
      jobs: [],
      logEntries: [],
      selectedBranchIndex: 0,
      selectedRunIndex: 0,
      selectedJobIndex: 0,
      activePanel: "branches",
      currentView: "main",
      statusMessage: ` ${resolvedOwner}/${resolvedRepo}  |  Poll: ${POLL_INTERVAL_MS / 1000}s`,
      errorMessage: null,
      loading: {
        branches: true,
        runs: true,
        jobs: false,
        logs: false,
      },
      pollIntervalMs: POLL_INTERVAL_MS,
    };

    const app = new App(state, ui);
    app.setupKeybindings();
    return app;
  }

  async start(): Promise<void> {
    this.setStatus("Starting lazyrun...");

    // Initial data load
    await this.loadBranches();
    if (this.state.branches.length > 0) {
      await this.loadRuns();
      if (this.state.runs.length > 0) {
        await this.loadJobs();
      }
    }

    // Start polling
    this.startPolling();

    // Focus first panel
    setActivePanel(
      this.ui.branchesList,
      this.ui.runsList,
      this.ui.detailsBox,
      this.state.activePanel,
    );
    this.focusActivePanel();

    this.render();
    this.setStatus(this.state.statusMessage);

    // Render and enter input loop
    this.ui.screen.render();
  }

  private setupKeybindings(): void {
    const { screen } = this.ui;

    // Global quit
    screen.key(["q", "C-c"], () => {
      this.stopPolling();
      process.exit(0);
    });

    // Tab cycle panels
    screen.key(["C-i"], () => {
      this.cyclePanel();
    });

    // Navigation
    screen.key(["up", "k"], () => {
      this.navigateUp();
    });

    screen.key(["down", "j"], () => {
      this.navigateDown();
    });

    screen.key(["home", "g"], () => {
      this.navigateHome();
    });

    screen.key(["end", "G"], () => {
      this.navigateEnd();
    });

    screen.key(["pageup"], () => {
      this.pageUp();
    });

    screen.key(["pagedown"], () => {
      this.pageDown();
    });

    // Enter to select
    screen.key(["return"], () => {
      this.handleEnter();
    });

    // Escape to go back / clear log view
    screen.key(["escape"], () => {
      if (this.state.currentView === "logs") {
        this.exitLogsView();
      }
    });

    // L to view logs
    screen.key(["l"], () => {
      if (this.state.activePanel === "runs" || this.state.activePanel === "details") {
        this.viewLogs();
      }
    });

    // R to refresh
    screen.key(["r"], () => {
      this.refresh();
    });

    // Switch to panel 1, 2, 3
    screen.key(["1"], () => {
      this.switchPanel("branches");
    });
    screen.key(["2"], () => {
      this.switchPanel("runs");
    });
    screen.key(["3"], () => {
      this.switchPanel("details");
    });

    // Handle window resize
    screen.on("resize", () => {
      this.render();
    });
  }

  private cyclePanel(): void {
    if (this.state.currentView === "logs") return; // No panel cycling in log view

    const panels: Panel[] = ["branches", "runs", "details"];
    const currentIndex = panels.indexOf(this.state.activePanel);
    const nextIndex = (currentIndex + 1) % panels.length;
    this.state.activePanel = panels[nextIndex]!;
    this.focusActivePanel();
    this.updatePanelStyles();
  }

  private switchPanel(panel: Panel): void {
    if (this.state.currentView === "logs") return;
    this.state.activePanel = panel;
    this.focusActivePanel();
    this.updatePanelStyles();
  }

  private focusActivePanel(): void {
    const { branchesList, runsList, detailsContent } = this.ui;
    switch (this.state.activePanel) {
      case "branches":
        branchesList.focus();
        break;
      case "runs":
        runsList.focus();
        break;
      case "details":
        detailsContent.focus();
        break;
    }
  }

  private navigateUp(): void {
    switch (this.state.activePanel) {
      case "branches":
        if (this.state.selectedBranchIndex > 0) {
          this.state.selectedBranchIndex--;
          this.onBranchSelected();
        }
        break;
      case "runs":
        if (this.state.selectedRunIndex > 0) {
          this.state.selectedRunIndex--;
          this.onRunSelected();
        }
        break;
    }
  }

  private navigateDown(): void {
    switch (this.state.activePanel) {
      case "branches":
        if (this.state.selectedBranchIndex < this.state.branches.length - 1) {
          this.state.selectedBranchIndex++;
          this.onBranchSelected();
        }
        break;
      case "runs":
        if (this.state.selectedRunIndex < this.state.runs.length - 1) {
          this.state.selectedRunIndex++;
          this.onRunSelected();
        }
        break;
    }
  }

  private navigateHome(): void {
    switch (this.state.activePanel) {
      case "branches":
        this.state.selectedBranchIndex = 0;
        this.onBranchSelected();
        break;
      case "runs":
        this.state.selectedRunIndex = 0;
        this.onRunSelected();
        break;
      case "details":
        this.ui.detailsContent.setScrollPerc(0);
        break;
    }
  }

  private navigateEnd(): void {
    switch (this.state.activePanel) {
      case "branches":
        this.state.selectedBranchIndex = this.state.branches.length - 1;
        this.onBranchSelected();
        break;
      case "runs":
        this.state.selectedRunIndex = this.state.runs.length - 1;
        this.onRunSelected();
        break;
      case "details":
        this.ui.detailsContent.setScrollPerc(100);
        break;
    }
  }

  private pageUp(): void {
    if (this.state.currentView === "logs" || this.state.activePanel === "details") {
      const content = this.ui.detailsContent;
      const scroll = content.getScroll();
      const height = content.height as number;
      content.scrollTo(Math.max(0, scroll - height));
    }
  }

  private pageDown(): void {
    if (this.state.currentView === "logs" || this.state.activePanel === "details") {
      const content = this.ui.detailsContent;
      const scroll = content.getScroll();
      const height = content.height as number;
      const maxScroll = content.getScrollHeight() - height;
      content.scrollTo(Math.min(maxScroll, scroll + height));
    }
  }

  private handleEnter(): void {
    switch (this.state.activePanel) {
      case "branches":
        // Branch already selected via navigate, but let's trigger explicit load
        this.loadRuns();
        break;
      case "runs":
        // Select run and view its jobs
        if (this.state.runs.length > 0) {
          this.viewLogs();
        }
        break;
      case "details":
        // If in log view, open the full log for the selected job
        this.viewLogs();
        break;
    }
  }

  private onBranchSelected(): void {
    updateBranchesList(
      this.ui.branchesList,
      this.state.branches,
      this.state.selectedBranchIndex,
      false,
    );
    this.loadRuns();
    this.render();
  }

  private onRunSelected(): void {
    if (this.state.activePanel === "runs" && this.state.selectedRunIndex < this.state.runs.length) {
      updateRunsList(
        this.ui.runsList,
        this.state.runs,
        this.state.selectedRunIndex,
        false,
        this.getCurrentBranchName(),
      );
      this.loadJobs();
      this.render();
    }
  }

  private async viewLogs(): Promise<void> {
    const run = this.getSelectedRun();
    if (!run) return;

    this.state.currentView = "logs";
    this.state.loading.logs = true;

    enterLogView(this.ui.detailsBox, this.ui.detailsContent);
    this.setStatus(`Loading logs for run #${run.runNumber}...`);
    this.render();

    try {
      const logs = await gh.getAllJobLogs(this.state.owner, this.state.repo, run.id);
      this.state.logEntries = logs;
      setLogContent(this.ui.detailsContent, logs);
      this.setStatus(`Logs for run #${run.runNumber}: ${logs.length} lines`);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      this.setStatus(`Error loading logs: ${msg}`);
      this.ui.detailsContent.setContent(`{red-fg}Error: ${msg}{/red-fg}`);
    } finally {
      this.state.loading.logs = false;
      this.render();
    }
  }

  private exitLogsView(): void {
    this.state.currentView = "main";
    this.state.logEntries = [];
    exitLogView(this.ui.detailsBox);
    this.loadJobs(); // Reload jobs into details panel
    this.updatePanelStyles();
    this.setStatus(this.state.statusMessage);
    this.render();
  }

  private async refresh(): Promise<void> {
    this.setStatus("Refreshing...");
    this.render();

    try {
      await this.loadBranches();
      if (this.state.branches.length > 0) {
        await this.loadRuns();
        if (this.state.runs.length > 0) {
          await this.loadJobs();
        }
      }
      this.setStatus("Refreshed ✓");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      this.setStatus(`Refresh failed: ${msg}`);
    }
    this.render();
  }

  // ── Data Loading ──

  private getCurrentBranchName(): string {
    const branch = this.state.branches[this.state.selectedBranchIndex];
    return branch?.name ?? "";
  }

  private getSelectedRun(): WorkflowRun | null {
    return this.state.runs[this.state.selectedRunIndex] ?? null;
  }

  private async loadBranches(): Promise<void> {
    this.state.loading.branches = true;
    this.updateBranchesUI();

    try {
      const branches = await gh.getBranches(this.state.owner, this.state.repo);
      this.state.branches = branches;
      if (this.state.selectedBranchIndex >= branches.length) {
        this.state.selectedBranchIndex = 0;
      }
      if (branches.length > 0 && this.state.selectedBranchIndex === 0) {
        // Auto-select default branch
        const defaultIdx = branches.findIndex((b) => b.isDefault);
        if (defaultIdx >= 0) {
          this.state.selectedBranchIndex = defaultIdx;
        }
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      this.setStatus(`Error loading branches: ${msg}`);
    } finally {
      this.state.loading.branches = false;
      this.updateBranchesUI();
    }
  }

  private async loadRuns(): Promise<void> {
    const branchName = this.getCurrentBranchName();
    if (!branchName) return;

    this.state.loading.runs = true;
    this.updateRunsUI();

    try {
      const runs = await gh.getRuns(this.state.owner, this.state.repo, branchName);
      this.state.runs = runs;
      if (this.state.selectedRunIndex >= runs.length) {
        this.state.selectedRunIndex = 0;
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      this.setStatus(`Error loading runs: ${msg}`);
    } finally {
      this.state.loading.runs = false;
      this.updateRunsUI();
    }
  }

  private async loadJobs(): Promise<void> {
    const run = this.getSelectedRun();
    if (!run) return;

    this.state.loading.jobs = true;
    this.updateDetailsUI();

    try {
      const jobs = await gh.getJobs(this.state.owner, this.state.repo, run.id);
      this.state.jobs = jobs;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      this.setStatus(`Error loading jobs: ${msg}`);
    } finally {
      this.state.loading.jobs = false;
      this.updateDetailsUI();
    }
  }

  // ── Polling ──

  private startPolling(): void {
    this.pollTimers.runs = setInterval(async () => {
      if (this.state.currentView !== "logs") {
        await this.loadRuns();
        await this.loadJobs();
        this.render();
      }
    }, this.state.pollIntervalMs);
  }

  private stopPolling(): void {
    if (this.pollTimers.runs) {
      clearInterval(this.pollTimers.runs);
      this.pollTimers.runs = null;
    }
  }

  // ── Rendering ──

  private render(): void {
    this.ui.screen.render();
  }

  private setStatus(msg: string): void {
    updateStatusBar(this.ui.statusBar, msg);
  }

  private updatePanelStyles(): void {
    setActivePanel(
      this.ui.branchesList,
      this.ui.runsList,
      this.ui.detailsBox,
      this.state.activePanel,
    );
  }

  private updateBranchesUI(): void {
    updateBranchesList(
      this.ui.branchesList,
      this.state.branches,
      this.state.selectedBranchIndex,
      this.state.loading.branches,
    );
  }

  private updateRunsUI(): void {
    updateRunsList(
      this.ui.runsList,
      this.state.runs,
      this.state.selectedRunIndex,
      this.state.loading.runs,
      this.getCurrentBranchName(),
    );
  }

  private updateDetailsUI(): void {
    const run = this.getSelectedRun();
    const text = buildRunDetailsText(run, this.state.jobs, this.state.loading.jobs);
    this.ui.detailsContent.setContent(text);
  }
}
