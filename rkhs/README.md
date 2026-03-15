# RKHS Explorer

Interactive visualizations of Reproducing Kernel Hilbert Space concepts using Go and [go-gg](https://github.com/aclements/go-gg).

## Quick start

```bash
go run .
# → Serving RKHS Explorer at http://localhost:8741
```

Opens a browser with five interactive plots:

1. **Kernel Functions** — RBF kernels centered at each sample
2. **Mean Embedding** — averaging kernel functions to represent a distribution
3. **MMD Comparison** — measuring distance between two distributions in the RKHS
4. **Gram Matrix** — pairwise kernel similarity heatmap
5. **Sigma Sweep** — how bandwidth affects the MMD measurement

Use the controls at the top to change distributions (P, Q), kernel bandwidth (σ), and sample count (n). Click **Resample** to draw fresh samples.

## Flags

```
-p        Distribution P (gaussian, bimodal, uniform, skewed) [default: bimodal]
-q        Distribution Q [default: gaussian]
-sigma    RBF kernel bandwidth [default: 0.5]
-n        Samples per distribution [default: 12]
-seed     Random seed [default: 42]
-port     HTTP server port [default: 8741]
-export   Write SVGs to disk instead of serving
-out      Output directory for -export [default: .]
```

## Examples

```bash
go run . -p skewed -q bimodal -sigma 0.3 -n 50
go run . -port 9000
go run . -export -out ./plots
```

`Ctrl+C` shuts down the server.
