"""Build a cute burgundy-and-gold rolled-carpet Windows icon using Pillow."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter
import math

ASSETS = Path(__file__).resolve().parents[1] / "assets"
ASSETS.mkdir(parents=True, exist_ok=True)
SIZE = 1024
shadow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
draw = ImageDraw.Draw(shadow)
draw.rounded_rectangle((27, 37, 993, 1003), radius=215, fill=(0, 0, 0, 90))
shadow = shadow.filter(ImageFilter.GaussianBlur(25))
image = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
image.alpha_composite(shadow)
d = ImageDraw.Draw(image)
d.rounded_rectangle((24, 20, 1000, 996), radius=215, fill="#581521", outline="#dcae6c", width=15)
d.rounded_rectangle((43, 39, 982, 977), radius=198, outline="#fff1cf", width=6)
d.rounded_rectangle((57, 53, 968, 961), radius=189, fill="#701a2b")

# A patterned carpet unrolling to the right.
carpet = [(352, 301), (850, 406), (872, 738), (340, 658)]
d.polygon(carpet, fill="#e8c58e")
d.line(carpet + [carpet[0]], fill="#c08a49", width=22, joint="curve")
d.polygon([(375,329),(817,422),(837,700),(370,620)], fill="#862435")
d.polygon([(388,344),(804,432),(823,678),(384,612)], fill="#9a3440")
for i in range(5):
    t = i / 4
    x = int(426 + 335 * t)
    top = int(360 + 87 * t)
    bottom = int(600 + 69 * t)
    d.line([(x, top), (x + 25, bottom)], fill="#e9bd73", width=10)
    for y in (top + 48, top + 106, top + 164):
        cx = x + 11
        d.polygon([(cx,y-18),(cx+18,y),(cx,y+18),(cx-18,y)], outline="#f6dfa5", width=6)
for x, y in [(440,406),(535,430),(630,459),(733,480),(454,554),(558,571),(661,603),(755,619)]:
    d.ellipse((x-7,y-7,x+7,y+7), fill="#f6dc98")
for y in range(438, 708, 30):
    d.line([(850,y),(900,y+7)], fill="#ffe1a7", width=9)

# The rolled end of the carpet, including a little spiral.
d.ellipse((125,260,482,704), fill=(38,7,13,190))
d.ellipse((137,267,481,679), fill="#b78146", outline="#ffebbb", width=16)
d.ellipse((154,284,463,661), fill="#7b2633", outline="#e4bc84", width=15)
d.ellipse((172,303,441,642), fill="#f3d4a1", outline="#6d1f2b", width=8)
d.ellipse((191,319,422,624), fill="#81303c", outline="#ffdbae", width=13)
points = []
for t in range(660):
    theta = t / 31
    radius = 10 + 6.3 * theta
    points.append((300 + math.cos(theta) * radius, 470 + math.sin(theta) * radius * 1.18))
d.line(points, fill="#f9ddb0", width=23, joint="curve")
d.ellipse((286,454,320,488), fill="#efbc72")
for x, y, r in [(744,270,19),(654,266,12),(818,307,11)]:
    d.line([(x-r,y),(x+r,y)], fill="#f7d69c", width=5)
    d.line([(x,y-r),(x,y+r)], fill="#f7d69c", width=5)

image = image.resize((512,512), Image.Resampling.LANCZOS)
image.save(ASSETS / "CarpetsAmeer.png", optimize=True)
image.save(ASSETS / "CarpetsAmeer.ico", format="ICO", sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)])
print("Carpets Ameer Windows icon generated")
