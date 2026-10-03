# Changelog

## [1.3.2](https://github.com/madflojo/tasks/compare/v1.3.1...v1.3.2) (2026-10-03)


### Bug Fixes

* **release:** include all configured changelog sections ([4395136](https://github.com/madflojo/tasks/commit/4395136a1b1bd851aaf9b6b5d2e1dfdecec3970c))
* **scheduler:** preserve replacements after RunOnce cleanup ([f40c92e](https://github.com/madflojo/tasks/commit/f40c92e01f053a35f89149ecf4c0eae64e2b175e))


### Tests

* **scheduler:** cover canceled delayed task scheduling - Testing Hunter ([a12b841](https://github.com/madflojo/tasks/commit/a12b841348857c702cd1c0e6dd1ae2ac3bda0cc2))

## [1.3.1](https://github.com/madflojo/tasks/compare/v1.3.0...v1.3.1) (2026-09-27)


### Bug Fixes

* **ci:** keep gh-aw runtime pins in sync 🧷 ([#63](https://github.com/madflojo/tasks/issues/63)) ([1588eb8](https://github.com/madflojo/tasks/commit/1588eb89159a73b115cc55702cb05e2cd3ce14b4))
* **ci:** preserve hunter rotation continuity ([d9b438f](https://github.com/madflojo/tasks/commit/d9b438f7cd6795c83ae4b82b830b9786c9787f96))
* **scheduler:** publish delayed timers under the task lock ([#69](https://github.com/madflojo/tasks/issues/69)) ([8b54b5d](https://github.com/madflojo/tasks/commit/8b54b5d03ce28c98e49a73739f60856ddd77bc5b))
* **workflows:** address Code Hunters review feedback ([3771dcf](https://github.com/madflojo/tasks/commit/3771dcf7523dd685a77e642432b647b4d4531e1c))
* **workflows:** address hunter review feedback 🐇 ([0bcc39c](https://github.com/madflojo/tasks/commit/0bcc39cada25f4c520c3eca18325bcbeced4ee94))
* **workflows:** keep hunter downloads and investigation on track ([a31f93e](https://github.com/madflojo/tasks/commit/a31f93e94c7d2d5fc569477575ad26f8145b9f40))


### Documentation

* **tasks:** fix non-compiling AddWithID godoc example ([#67](https://github.com/madflojo/tasks/pull/67)) ([ab4fc9d](https://github.com/madflojo/tasks/commit/ab4fc9d0a909749f31300fdf0e2317bed51b32ff))

### Code Refactoring

* **scheduler:** consolidate duplicated timer arming logic ([#65](https://github.com/madflojo/tasks/pull/65)) ([cc8762d](https://github.com/madflojo/tasks/commit/cc8762d640cef585cc76f98624db663cf853671f))

### Tests

* **scheduler:** cover Del no-op and idempotent paths ([23c6897](https://github.com/madflojo/tasks/commit/23c68978725bf92fb6a3e043e94ef437ca390872))

### Continuous Integration

* **deps:** bump codecov/codecov-action from 7.1.0 to 7.1.1 ([#68](https://github.com/madflojo/tasks/pull/68)) ([31974bd](https://github.com/madflojo/tasks/commit/31974bd33cb5c353f3ada0d577927b9425446fa7))
* **deps:** bump codecov/codecov-action from 7.0.0 to 7.1.0 ([#64](https://github.com/madflojo/tasks/pull/64)) ([fc57423](https://github.com/madflojo/tasks/commit/fc57423adaf7b4bc1be7adf87f79a1e4904f23fc))
* **deps:** bump github/gh-aw-actions/setup from 0.86.2 to 0.88.0 ([3bb3e83](https://github.com/madflojo/tasks/commit/3bb3e83b7ef487379e46dab60cfd4c0e0450d21a))
* **code-hunters:** schedule rotating maintenance hunts ([0b76c13](https://github.com/madflojo/tasks/commit/0b76c1333f106a65dab3a2a4b7fa14afdd9d785f))
* **deps:** bump actions/checkout from 7.0.0 to 7.0.1 ([1b98cdd](https://github.com/madflojo/tasks/commit/1b98cdd2d8ae9796b454409f4256b3a94d9d18d0))
* **deps:** bump actions/setup-go from 6.4.0 to 7.0.0 ([a1a03f3](https://github.com/madflojo/tasks/commit/a1a03f31f0a1acca64b66eabaf3479ebcfd6faa8))
* **deps:** bump golangci/golangci-lint-action from 9.2.1 to 9.3.0 ([35e5b3f](https://github.com/madflojo/tasks/commit/35e5b3f485138d2ff7d45ff325579b797c362aec))
* **deps:** bump actions/checkout from 6.0.3 to 7.0.0 ([17210fe](https://github.com/madflojo/tasks/commit/17210fe5792da5c87f73a6d8a969f35e1572a236))
* **deps:** bump codecov/codecov-action from 6.0.1 to 7.0.0 ([6976241](https://github.com/madflojo/tasks/commit/697624178246e50299b4c6e2263d3c62e255af79))
* **deps:** bump actions/checkout from 6.0.2 to 6.0.3 ([f52bd6a](https://github.com/madflojo/tasks/commit/f52bd6a0e0ecdcf894065e2f0275cc06770b6a18))
* **deps:** bump codecov/codecov-action from 6.0.0 to 6.0.1 ([0229312](https://github.com/madflojo/tasks/commit/0229312103987e2359301bdcb1f7223939335322))
* **deps:** bump golangci/golangci-lint-action from 9.2.0 to 9.2.1 ([f23048e](https://github.com/madflojo/tasks/commit/f23048eba3eb9159cf160dfca6f6f8385b56ca25))

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
