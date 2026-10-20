import argparse
import gc
from pathlib import Path

import numpy as np
import torch
from PIL import Image, ImageOps


PREFERRED_EXTS = (".png", ".jpg", ".jpeg", ".webp", ".avif")


def choose_inputs(input_dir: Path) -> list[Path]:
    grouped: dict[str, list[Path]] = {}
    for path in input_dir.iterdir():
        if path.is_file() and path.suffix.lower() in PREFERRED_EXTS:
            grouped.setdefault(path.stem, []).append(path)

    picked = []
    for stem, paths in grouped.items():
        paths.sort(key=lambda p: PREFERRED_EXTS.index(p.suffix.lower()))
        picked.append(paths[0])
    return sorted(picked)


def normalize_depth(depth: np.ndarray, invert: bool) -> Image.Image:
    depth = np.asarray(depth, dtype=np.float32)
    depth = np.nan_to_num(depth)
    lo = float(np.percentile(depth, 1.0))
    hi = float(np.percentile(depth, 99.0))
    if hi <= lo:
        lo = float(depth.min())
        hi = float(depth.max())
    if hi <= lo:
        norm = np.zeros_like(depth, dtype=np.float32)
    else:
        norm = np.clip((depth - lo) / (hi - lo), 0.0, 1.0)
    if invert:
        norm = 1.0 - norm
    return Image.fromarray((norm * 255.0 + 0.5).astype(np.uint8), mode="L")


def extract_depth(output) -> np.ndarray:
    for attr in ("depth_np", "prediction", "pred_np"):
        value = getattr(output, attr, None)
        if value is not None:
            return np.asarray(value)
    if isinstance(output, dict):
        for key in ("depth_np", "prediction", "pred_np"):
            if key in output:
                return np.asarray(output[key])
    raise RuntimeError(f"Cannot find depth array in Marigold output: {type(output)!r}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", default="", help="Single image to process. Overrides --input-dir.")
    parser.add_argument("--input-dir", default="usr/uploads/background")
    parser.add_argument("--output-dir", default="usr/uploads/depth-marigold")
    parser.add_argument("--output", default="", help="Single output path. Only valid with --input.")
    parser.add_argument("--model", default="prs-eth/marigold-depth-v1-1")
    parser.add_argument("--processing-res", type=int, default=768)
    parser.add_argument("--steps", type=int, default=10)
    parser.add_argument("--ensemble-size", type=int, default=5)
    parser.add_argument("--invert", action="store_true")
    parser.add_argument("--overwrite", action="store_true")
    parser.add_argument("--limit", type=int, default=0)
    args = parser.parse_args()

    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    if args.input:
        inputs = [Path(args.input)]
    else:
        input_dir = Path(args.input_dir)
        inputs = choose_inputs(input_dir)
    if args.limit:
        inputs = inputs[: args.limit]
    if not inputs:
        raise SystemExit(f"No images found under {input_dir}")

    from diffusers import MarigoldDepthPipeline

    device = "cuda" if torch.cuda.is_available() else "cpu"
    dtype = torch.float16 if device == "cuda" else torch.float32
    if device == "cuda":
        torch.backends.cuda.matmul.allow_tf32 = True

    pipe = MarigoldDepthPipeline.from_pretrained(
        args.model,
        torch_dtype=dtype,
        variant="fp16" if dtype == torch.float16 else None,
    )
    pipe = pipe.to(device)
    if hasattr(pipe, "enable_attention_slicing"):
        pipe.enable_attention_slicing()

    print(f"model={args.model} device={device} count={len(inputs)}")
    for idx, src in enumerate(inputs, 1):
        dst = Path(args.output) if args.output else output_dir / f"{src.stem}.png"
        dst.parent.mkdir(parents=True, exist_ok=True)
        if dst.exists() and not args.overwrite:
            print(f"[{idx}/{len(inputs)}] skip {dst}")
            continue

        image = ImageOps.exif_transpose(Image.open(src)).convert("RGB")
        print(f"[{idx}/{len(inputs)}] {src} -> {dst} ({image.width}x{image.height})")
        with torch.inference_mode():
            output = pipe(
                image,
                num_inference_steps=args.steps,
                ensemble_size=args.ensemble_size,
                processing_res=args.processing_res,
                match_input_res=True,
            )
        depth = extract_depth(output)
        normalize_depth(depth, args.invert).save(dst, optimize=True)

        del output, depth, image
        if device == "cuda":
            torch.cuda.empty_cache()
        gc.collect()


if __name__ == "__main__":
    main()
