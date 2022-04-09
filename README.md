# memeFSGo - A filesystem whose contents are memes from reddit ![GitHub Actions Workflow](https://github.com/plsmaop/memeFSGo/workflows/CI/badge.svg)
*Insipred by [memefs](https://github.com/svenstaro/memefs)*

## Dependency
- FUSE
  - Ubuntu: 
    ```bash
    sudo apt-get install libfuse-dev
    ```
  - MACOSX: [osxfuse](https://osxfuse.github.io/)

## Build
```bash
go build .
```

## Execute
```bash
mkdir memes
./memefsGo -m memes
```

## Help
```
./memefsGo --help
```

## TODO
- [ ] Test
- [ ] Documentation
- [ ] Thread-safe
- [ ] Makefile
- [ ] Github release
- [ ] Bump version
- [ ] Windows
