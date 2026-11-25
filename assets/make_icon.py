# Рисует иконку «Фокуса»: assets/icon.png (1024) и assets/icon.ico. Запускается при сборке.
from PIL import Image, ImageDraw
S = 1024; sc = 4; N = S * sc
g = Image.new('RGBA', (N, N)); d = ImageDraw.Draw(g)
a = (99, 102, 241); c = (55, 48, 163)
for y in range(N):
    t = y / N
    d.line([(0, y), (N, y)], fill=tuple(int(a[i] * (1 - t) + c[i] * t) for i in range(3)) + (255,))
mask = Image.new('L', (N, N), 0)
ImageDraw.Draw(mask).rounded_rectangle([100 * sc, 100 * sc, N - 100 * sc, N - 100 * sc], radius=230 * sc, fill=255)
img = Image.new('RGBA', (N, N), (0, 0, 0, 0)); img.paste(g, (0, 0), mask)
d = ImageDraw.Draw(img); cx = cy = N // 2; W = (255, 255, 255, 255); w = 46 * sc
R1 = 300 * sc; d.ellipse([cx - R1, cy - R1, cx + R1, cy + R1], outline=W, width=w)
R2 = 170 * sc; d.ellipse([cx - R2, cy - R2, cx + R2, cy + R2], outline=(255, 255, 255, 200), width=w)
R3 = 62 * sc; d.ellipse([cx - R3, cy - R3, cx + R3, cy + R3], fill=W)
for (x0, y0, x1, y1) in [(cx, cy - R1 - 70 * sc, cx, cy - R1 + 40 * sc), (cx, cy + R1 - 40 * sc, cx, cy + R1 + 70 * sc),
                         (cx - R1 - 70 * sc, cy, cx - R1 + 40 * sc, cy), (cx + R1 - 40 * sc, cy, cx + R1 + 70 * sc, cy)]:
    d.line([(x0, y0), (x1, y1)], fill=W, width=w)
img = img.resize((S, S), Image.LANCZOS)
img.save('assets/icon.png')
img.save('assets/icon.ico', sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
