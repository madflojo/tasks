# Changelog

## [1.3.0](https://github.com/madflojo/tasks/compare/v1.2.1...v1.3.0) (2026-05-10)


### Features

* recover panics from task callbacks ([62a350c](https://github.com/madflojo/tasks/commit/62a350c74101782b2ff43fdaedd5e34d4ec4af8e))


### Bug Fixes

* avoid nil recover panic reports ([321b1fc](https://github.com/madflojo/tasks/commit/321b1fcb25f0269d5547378da6563e244fad7cd2))
* **ci:** remove stale goveralls install ([00f2bf7](https://github.com/madflojo/tasks/commit/00f2bf7d85b62a1ada421625ff773a43fbbf618c))
* guard delayed interval scheduling ([32f6a56](https://github.com/madflojo/tasks/commit/32f6a5664af505b0d4acc4ae0acc6ea7936ec137))
* make scheduler errors branchable ([3c4465b](https://github.com/madflojo/tasks/commit/3c4465b8ca63e9ce846f4dfc9effaf7309148ab4))
* make scheduler errors branchable ([75fb505](https://github.com/madflojo/tasks/commit/75fb5053ad93a2bb36b0120544d903a2ac0e2856))
* recover panics from task callbacks ([fb0a68c](https://github.com/madflojo/tasks/commit/fb0a68cc209406f7eba23927af8fb5318c1aa450))
* reject empty custom task IDs ([ce2cbfc](https://github.com/madflojo/tasks/commit/ce2cbfc10bb33c10e5ea2fdfdf50a2755f61a4f1))
* reject empty custom task IDs ([9f8cb02](https://github.com/madflojo/tasks/commit/9f8cb02238b07739761216426912154cc6661187))
* report nil task panics ([90002ad](https://github.com/madflojo/tasks/commit/90002ad86894dacd4208dce3e6d539d249f58e5d))
* **scheduler:** avoid mutating caller tasks on duplicate IDs ([4894e67](https://github.com/madflojo/tasks/commit/4894e67e8c72a16931b609e83846924897ebc095))
* **scheduler:** avoid mutating caller tasks on duplicate IDs ([c2ab9b4](https://github.com/madflojo/tasks/commit/c2ab9b4dce382e4e18fd2956e98978da64c8f217))
* **scheduler:** clear runtime state on cloned task before scheduling ([78d06b2](https://github.com/madflojo/tasks/commit/78d06b2aa16347e5545412bb2b147e85032827c0))
* **scheduler:** harden task validation and add repo make targets ([47b4e8b](https://github.com/madflojo/tasks/commit/47b4e8b5901173a015f40ce00f6582a9ce42062b))
* **scheduler:** harden task validation and add repo make targets ([30bce88](https://github.com/madflojo/tasks/commit/30bce88e00d82424e703614e5d14f2e07b5912a6))
* **scheduler:** stop canonical tasks after delete ([325a5d7](https://github.com/madflojo/tasks/commit/325a5d7276374b21592883e4300c5eda78820825))
* **scheduler:** stop canonical tasks after delete ([73f26fe](https://github.com/madflojo/tasks/commit/73f26fed9d2135356449df755c4cd2f508a634cf))
* stop delayed start timers ([9de83a7](https://github.com/madflojo/tasks/commit/9de83a73c3ab039799aed39542e726957c7cb6e3))
* stop delayed start timers ([e58a966](https://github.com/madflojo/tasks/commit/e58a9662c08cfcf6f4e601d01d6667f9748187d2))
* **test:** use errCtx/ErrFunc instead of t.Errorf in TaskFunc goroutine ([4c1b4b7](https://github.com/madflojo/tasks/commit/4c1b4b7d5e659c18e0726e4f4e055eb96b477d45))
