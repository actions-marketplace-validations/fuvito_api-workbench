"""Generate appicon.png and windows/icon.ico for ApiDesk."""
import os
from PIL import Image, ImageDraw

BG      = (9, 9, 11, 255)        # #09090b
FG      = (244, 244, 245, 255)   # #f4f4f5
ACCENT  = (59, 130, 246, 255)    # #3b82f6

def draw_icon(size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    s = size / 256  # scale factor

    # Rounded background
    r = int(52 * s)
    d.rounded_rectangle([0, 0, size - 1, size - 1], radius=r, fill=BG)

    def rect(x, y, w, h, color):
        rr = max(1, int(3 * s))
        x0, y0 = int(x * s), int(y * s)
        x1, y1 = max(x0 + 1, int((x + w) * s) - 1), max(y0 + 1, int((y + h) * s) - 1)
        if x1 <= x0 or y1 <= y0:
            return
        d.rounded_rectangle([x0, y0, x1, y1], radius=rr, fill=color)

    # A glyph: white sides + mid crossbar, blue top cap (drawn last to overlay)
    rect(42,  56, 14, 144, FG)     # left bar
    rect(100, 56, 14, 144, FG)     # right bar
    rect(42,  121, 72, 14, FG)     # mid crossbar
    rect(42,  56,  72, 14, ACCENT) # blue top cap

    # D glyph: white letterform, blue right bar
    rect(138, 56,  14, 144, FG)    # stem
    rect(138, 56,  76,  14, FG)    # top bar
    rect(138, 186, 76,  14, FG)    # bottom bar
    rect(200, 56,  14, 144, ACCENT) # blue right bar

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
