package resources

type Skill struct {
	Name        string
	Trend       string // "+12%", "-3%", "—"
	Up          bool
	Score       int
	Category    string
	Description string
}

type Repo struct {
	Owner       string
	Name        string
	Description string
	Stars       int
	Forks       int
	Language    string
	URL         string
}

func seedTrending() []Skill {
	return []Skill{
		{"bun", "+47%", true, 9120, "runtime", "the node-killer that still hasn't killed node but is trying"},
		{"zod-3", "+38%", true, 7204, "validation", "runtime types because typescript can't be trusted alone"},
		{"htmx", "+33%", true, 8431, "frontend", "the html-is-fine framework for people tired of frameworks"},
		{"sqlite-vec", "+28%", true, 4112, "embeddings", "vectors in sqlite, because you didn't need a vector db"},
		{"tauri-2", "+24%", true, 6801, "desktop", "electron but it doesn't eat 800mb of ram to render a button"},
		{"effect-ts", "+19%", true, 3920, "fp", "the functional library that explains itself in 47 medium posts"},
		{"valkey", "+18%", true, 5102, "cache", "redis but for people with feelings about licenses"},
		{"deno-2", "+15%", true, 4881, "runtime", "still trying, still kind of working"},
		{"jujutsu", "+13%", true, 2104, "vcs", "git but with fewer footguns and more confused devs"},
		{"astro-5", "+11%", true, 6712, "ssg", "ship html, get medals"},
		{"rspack", "+9%", true, 3204, "bundler", "webpack in rust, for when you have 47 entry points"},
		{"biome", "+8%", true, 4109, "tooling", "linter + formatter that finally agrees with itself"},
		{"webgpu", "+6%", true, 2410, "gpu", "the graphics api the browser actually deserves"},
		{"oxc", "+4%", true, 1820, "tooling", "js tooling in rust, because everything must be in rust"},
		{"react-19", "-2%", false, 12044, "frontend", "the framework you can't quit even when you want to"},
	}
}

func seedTop() []Skill {
	return []Skill{
		{"typescript", "—", true, 99412, "language", "javascript with feelings about correctness"},
		{"react", "—", true, 92044, "frontend", "the framework that won and now everyone resents"},
		{"postgres", "—", true, 87120, "database", "the answer to every database question, including the wrong ones"},
		{"docker", "—", true, 81002, "infra", "it works on your container, ship the container"},
		{"kubernetes", "—", true, 74301, "infra", "1000 yaml files in a trench coat pretending to be infra"},
		{"rust", "—", true, 71204, "language", "your borrow checker is the senior engineer you deserve"},
		{"go", "—", true, 68210, "language", "if err != nil { return nil, err }. and again. and again."},
		{"nginx", "—", true, 65120, "web", "reverse-proxy your problems away"},
		{"redis", "—", true, 61420, "cache", "the in-memory store that's also a queue, db, lock, oracle"},
		{"vim", "—", true, 58910, "editor", "you'll figure out how to quit eventually. or not."},
	}
}

func seedRepos() []Repo {
	return []Repo{
		{"oven-sh", "bun", "Incredibly fast JavaScript runtime, bundler, transpiler, and package manager",
			78201, 2940, "Zig", "https://github.com/oven-sh/bun"},
		{"ggerganov", "llama.cpp", "Inference of LLaMA models in pure C/C++",
			69210, 9810, "C++", "https://github.com/ggerganov/llama.cpp"},
		{"jj-vcs", "jj", "A Git-compatible VCS that is both simple and powerful",
			18402, 612, "Rust", "https://github.com/jj-vcs/jj"},
		{"astral-sh", "uv", "An extremely fast Python package and project manager, written in Rust",
			34102, 920, "Rust", "https://github.com/astral-sh/uv"},
		{"biomejs", "biome", "A toolchain for web projects, aimed to provide functionalities to maintain them",
			17820, 540, "Rust", "https://github.com/biomejs/biome"},
		{"htmx-org", "htmx", "</> htmx - high power tools for HTML",
			41210, 1402, "JavaScript", "https://github.com/htmx-org/htmx"},
		{"tauri-apps", "tauri", "Build smaller, faster, and more secure desktop applications with a web frontend",
			84102, 2540, "Rust", "https://github.com/tauri-apps/tauri"},
		{"colinhacks", "zod", "TypeScript-first schema validation with static type inference",
			37210, 1320, "TypeScript", "https://github.com/colinhacks/zod"},
		{"withastro", "astro", "The web framework for content-driven websites",
			47102, 2410, "TypeScript", "https://github.com/withastro/astro"},
		{"charmbracelet", "bubbletea", "A powerful little TUI framework",
			28412, 880, "Go", "https://github.com/charmbracelet/bubbletea"},
		{"sqlite", "sqlite", "Official Git mirror of the SQLite source tree",
			7102, 410, "C", "https://github.com/sqlite/sqlite"},
		{"valkey-io", "valkey", "A flexible distributed key-value datastore, BSD-licensed",
			19204, 720, "C", "https://github.com/valkey-io/valkey"},
	}
}
