"""Logo kaynağından (branding/logo_source.png) simge ve açılış görsellerini üretir.

Çalıştırma (apps/mobile içinden):  python branding/make_icons.py
Sonra:  dart run flutter_launcher_icons  ve  dart run flutter_native_splash:create
Gerekli: Pillow (pip install pillow).
"""
import os
import sys
from PIL import Image, ImageDraw, ImageFilter
HERE = os.path.dirname(os.path.abspath(__file__))
src = Image.open(sys.argv[1] if len(sys.argv) > 1 else os.path.join(HERE, 'logo_source.png')).convert('RGBA')
out = lambda name: os.path.join(HERE, name)
BG = (3, 2, 2, 255)
# İçerik kutusu (parlaklık > 60): x 61–1458, y 0–942; biraz pay.
box = (40, 0, 1496, 960)
content = src.crop(box)
cw, ch = content.size

def feather(img, edge=0.10):
    """Kenarlara doğru yumuşak saydamlık (dikdörtgen kesik görünmesin)."""
    w, h = img.size
    mask = Image.new('L', (w, h), 0)
    d = ImageDraw.Draw(mask)
    ex, ey = int(w*edge), int(h*edge)
    d.rounded_rectangle((ex, ey, w-ex, h-ey), radius=min(ex, ey)*2, fill=255)
    mask = mask.filter(ImageFilter.GaussianBlur(min(ex, ey)*0.8))
    out = img.copy(); out.putalpha(mask); return out

def square(size, width_ratio, transparent=False):
    canvas = Image.new('RGBA', (size, size), (0,0,0,0) if transparent else BG)
    w = int(size*width_ratio); h = int(w*ch/cw)
    piece = feather(content.resize((w, h), Image.LANCZOS))
    canvas.alpha_composite(piece, ((size-w)//2, (size-h)//2))
    return canvas

# 1) Play Store simgesi 512 ve genel kaynak 1024 (tam kare, düz arka plan)
square(1024, 1.0).convert('RGB').save(out('icon_1024.png'))
square(512, 1.0).convert('RGB').save(out('play_icon_512.png'))
# 2) Uyarlanabilir simge ön planı: 108dp tuval, görünür güvenli daire 66dp.
#    İçerik oranı ~0.66 → genişlik ≈ 0.55 × 108dp (köşeler daire içinde kalır)
square(1024, 0.72, transparent=True).save(out('adaptive_fg_1024.png'))
# 3) Açılış ekranı (Android 12+ simge alanı da daire): ön planla aynı
# Önizlemeler: daire maskeli 48dp ve 192dp
def masked_preview(fg, size):
    bg = Image.new('RGBA', fg.size, BG); bg.alpha_composite(fg)
    # 108dp tuvalin ortadaki 72dp'si görünür
    s = fg.size[0]; inset = int(s*18/108)
    vis = bg.crop((inset, inset, s-inset, s-inset)).resize((size, size), Image.LANCZOS)
    m = Image.new('L', (size, size), 0); ImageDraw.Draw(m).ellipse((0,0,size-1,size-1), fill=255)
    out = Image.new('RGBA', (size, size), (230,230,230,255)); out.paste(vis, (0,0), m); return out
fg = Image.open(out('adaptive_fg_1024.png'))
prev = Image.new('RGBA', (560, 240), (230,230,230,255))
prev.alpha_composite(masked_preview(fg, 192), (10, 20))
prev.alpha_composite(masked_preview(fg, 96), (220, 68))
prev.alpha_composite(masked_preview(fg, 48), (330, 92))
store = Image.open(out('play_icon_512.png')).convert('RGBA').resize((180,180))
m = Image.new('L', (180,180), 0); ImageDraw.Draw(m).rounded_rectangle((0,0,179,179), radius=36, fill=255)
prev.paste(store, (370, 30), m)
prev.convert("RGB").save(out("preview.png")) if os.environ.get("PREVIEW") else None

# 4) Android 12 öncesi açılış görseli: tam logo, 4x yoğunlukta ~280dp genişlik
w = 1120; h = int(w*ch/cw)
feather(content.resize((w, h), Image.LANCZOS)).save(out('splash_logo.png'))
# 5) Uygulama içi logo (açılış ekranı): WebP, şeffaf kenarlı
app = feather(content.resize((900, int(900*ch/cw)), Image.LANCZOS))
app.save(os.path.join(HERE, '..', 'assets', 'images', 'logo.webp'), 'WEBP', quality=88, method=6)
