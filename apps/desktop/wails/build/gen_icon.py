"""Generate appicon.png and windows/icon.ico for Clear Signal."""
import os
from PIL import Image, ImageDraw

BG      = (9, 9, 11, 255)        # #09090b
FG      = (244, 244, 245, 255)   # #f4f4f5
ACCENT  = (59, 130, 246, 255)    # #3b82f6

# Signal bars defined at 256px scale: (x, y, width, height, color)
BARS = [
    (48,  156, 28,  50, FG),
    (92,  116, 28,  90, FG),
    (136,  86, 28, 120, FG),
    (180,  56, 28, 150, ACCENT),
]

def draw_icon(size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    s = size / 256  # scale factor

    # Rounded background
    r = int(52 * s)
    d.rounded_rectangle([0, 0, size - 1, size - 1], radius=r, fill=BG)

    for bx, by, bw, bh, color in BARS:
        rr = max(1, int(4 * s))
        x0, y0 = int(bx * s), int(by * s)
        x1 = max(x0 + 1, int((bx + bw) * s) - 1)
        y1 = max(y0 + 1, int((by + bh) * s) - 1)
        if x1 <= x0 or y1 <= y0:
            continue
        d.rounded_rectangle([x0, y0, x1, y1], radius=rr, fill=color)

    return img


def main():
    here = os.path.dirname(__file__)

    # appicon.png — 256×256
    icon256 = draw_icon(256)
    icon256.save(os.path.join(here, "appicon.png"), "PNG")
    print("wrote appicon.png")

    # windows/icon.ico — multi-size
    win_dir = os.path.join(here, "windows")
    os.makedirs(win_dir, exist_ok=True)
    sizes = [16, 24, 32, 48, 64, 128, 256]
    frames = [draw_icon(sz).convert("RGBA") for sz in sizes]
    frames[0].save(
        os.path.join(win_dir, "icon.ico"),
        format="ICO",
        sizes=[(sz, sz) for sz in sizes],
        append_images=frames[1:],
    )
    print("wrote windows/icon.ico")


if __name__ == "__main__":
    main()
