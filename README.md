# Brainf**k JIT compiler in Golang
Brainf**kのJITコンパイラをGoで実装してみた。
元ネタは[これ](https://postd.cc/adventures-in-jit-compilation-part-1-an-interpreter/)（C++実装）

動作環境はUbuntu 18.04

一番高速に実行できるのは`cmd/optjit/optjit.go`

```sh
go build cmd/optjit/optjit.go

time echo 179424691 | ./optjit factor.bf
time ./optjit mandelbrot.bf
```
