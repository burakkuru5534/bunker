"""Senaryo tanıtımı kaynaklarını (branding/scenarios) uygulama varlıklarına çevirir.

Çalıştırma (apps/mobile içinden):  python branding/scenarios/process_scenarios.py
Gerekli: Pillow, numpy, scipy (pip install pillow numpy scipy) ve PATH'te ffmpeg.

Çıktı: assets/scenarios/<id>/{back,mid,front,hero}.webp, narration.ogg, impact.ogg,
       assets/scenarios/door.webp, music.ogg

- back / door: olduğu gibi WebP'ye.
- mid / front / hero: yeşil zemin silinir (zemin rengi kenardan ölçülür,
  renk farkıyla yumuşak saydamlık, kenara bağlı bölgeler, "unmix" ile
  yeşil yansımanın geri alınması). Parlayan nesnelerde (patlama, göktaşı,
  güneş) yeşil/zeytin hale ayrıca temizlenir.
- Sesler: Ogg/Opus; anlatım mono 48 kbps, efekt ve müzik stereo 96 kbps,
  ses düzeyi EBU R128'e göre eşitlenir (anlatım -16, efekt -14, müzik -24 LUFS).
"""
import os
import subprocess

import numpy as np
from PIL import Image
from scipy import ndimage

HERE = os.path.dirname(os.path.abspath(__file__))
ASSETS = os.path.join(HERE, '..', '..', 'assets', 'scenarios')
SCENARIOS = ['nuclear_winter', 'red_cough', 'meteor', 'silent_machines', 'solar_storm']
LAYERS = ['mid', 'front', 'hero']

# Zemin bağlantısı: 'top' = üst kenar (+ yanların üst %40'ı), 'all' = dört
# kenar, 'global' = bağlantı aranmaz (zemin rengi içerikte hiç yoksa).
SEEDS = {('solar_storm', 'mid'): 'global'}
# Zemin sapması üst sınırı; patlama görselinde alt arazi de silinsin diye yok.
SPREAD_CAP = {('nuclear_winter', 'hero'): None}
GLOWING = {'nuclear_winter', 'meteor', 'solar_storm'}
# Kaynak görseldeki imza/filigran alanları (x0, y0, x1, y1; oran) → saydam.
ERASE = {('meteor', 'hero'): (0.0, 0.85, 0.2, 1.0)}


def _src(scenario, name):
    for ext in ('png', 'jpg', 'jpeg', 'webp'):
        p = os.path.join(HERE, scenario, f'{name}.{ext}') if scenario else os.path.join(HERE, f'{name}.{ext}')
        if os.path.exists(p):
            return p
    raise FileNotFoundError(f'{scenario}/{name}')


def chroma_key(path, seeds, spread_cap=25.0):
    im = np.asarray(Image.open(path).convert('RGB')).astype(np.float32)
    h, w, _ = im.shape
    if seeds == 'all':
        samp = np.concatenate([im[:6].reshape(-1, 3), im[-6:].reshape(-1, 3),
                               im[:, :6].reshape(-1, 3), im[:, -6:].reshape(-1, 3)])
    else:
        samp = im[:int(h * 0.06)].reshape(-1, 3)
    bg = np.median(samp, 0)
    if spread_cap is None:
        # Kenarlar karışıksa (ör. patlama görselinin alt arazisi) geniş eşik.
        spread = float(np.percentile(np.linalg.norm(samp - bg, axis=1), 95))
        t0, t1 = spread + 6, spread + 44
    else:
        spread = min(float(np.percentile(np.linalg.norm(samp - bg, axis=1), 90)), spread_cap)
        t0, t1 = spread + 8, spread + 53
    d = np.linalg.norm(im - bg, axis=2)
    a = np.clip((d - t0) / (t1 - t0), 0, 1)
    if seeds != 'global':
        lab, _ = ndimage.label(d < t1)
        if seeds == 'top':
            border = np.concatenate([lab[0], lab[:int(h * 0.4), 0], lab[:int(h * 0.4), -1]])
        else:
            border = np.concatenate([lab[0], lab[-1], lab[:, 0], lab[:, -1]])
        conn = np.isin(lab, list(set(np.unique(border)) - {0}))
        a = np.where(conn, a, 1.0)
    a = ndimage.gaussian_filter(a, 0.6 if spread_cap is not None else 0.7)
    aa = np.maximum(a, 1e-3)[..., None]
    fg = np.where(aa > 0.02, (im - (1 - aa) * bg) / aa, im).clip(0, 255)
    g, lim = fg[..., 1], np.maximum(fg[..., 0], fg[..., 2]) * 1.05
    fg[..., 1] = np.where((a < 0.98) & (g > lim), lim, g)
    return np.dstack([fg, a * 255])


def clean_glow(rgba):
    f, al = rgba[..., :3], rgba[..., 3] / 255
    green = (f[..., 1] - np.maximum(f[..., 0], f[..., 2])) / 255
    olive = (f[..., 1] - f[..., 0] * 0.85) / 255 * np.clip((200 - f[..., 0]) / 60, 0, 1)
    al = al * (1 - np.clip(np.maximum(green * 5, olive * 3), 0, 1))
    edge = al < 0.995
    f[..., 1] = np.where(edge, np.minimum(f[..., 1], f[..., 0] * 0.8), f[..., 1])
    f[..., 2] = np.where(edge, np.minimum(f[..., 2], f[..., 0] * 0.6), f[..., 2])
    rgba[..., 3] = al * 255
    return rgba


def layer(scenario, name):
    key = (scenario, name)
    rgba = chroma_key(_src(scenario, name), SEEDS.get(key, 'all' if name == 'hero' else 'top'),
                      SPREAD_CAP.get(key, 25.0))
    if name == 'hero' and scenario in GLOWING:
        rgba = clean_glow(rgba)
    if key in ERASE:
        h, w = rgba.shape[:2]
        x0, y0, x1, y1 = ERASE[key]
        rgba[int(h * y0):int(h * y1), int(w * x0):int(w * x1), 3] = 0
    return Image.fromarray(rgba.clip(0, 255).astype(np.uint8), 'RGBA')


def audio(src, dst, *, mono, kbps, lufs, max_seconds=None):
    args = ['ffmpeg', '-y', '-loglevel', 'error', '-i', src, '-vn', '-map_metadata', '-1']
    if max_seconds:
        args += ['-t', str(max_seconds)]
    args += ['-af', f'loudnorm=I={lufs}:TP=-1.5:LRA=11', '-ar', '48000',
             '-ac', '1' if mono else '2', '-c:a', 'libopus', '-b:a', f'{kbps}k', dst]
    subprocess.run(args, check=True)


def main():
    os.makedirs(ASSETS, exist_ok=True)
    Image.open(_src(None, 'door')).convert('RGB').save(os.path.join(ASSETS, 'door.webp'), quality=80, method=6)
    audio(_audio_src(None, 'music_loop'),
          os.path.join(ASSETS, 'music.ogg'), mono=False, kbps=96, lufs=-24, max_seconds=30)
    for s in SCENARIOS:
        out = os.path.join(ASSETS, s)
        os.makedirs(out, exist_ok=True)
        Image.open(_src(s, 'back')).convert('RGB').save(os.path.join(out, 'back.webp'), quality=80, method=6)
        for n in LAYERS:
            layer(s, n).save(os.path.join(out, f'{n}.webp'), quality=85, method=6)
        audio(_audio_src(s, 'narration'), os.path.join(out, 'narration.ogg'), mono=True, kbps=48, lufs=-16)
        audio(_audio_src(s, 'impact'), os.path.join(out, 'impact.ogg'), mono=False, kbps=96, lufs=-14)
        print('ok', s)


def _audio_src(scenario, name):
    base = os.path.join(HERE, scenario) if scenario else HERE
    for ext in ('wav', 'mp3', 'ogg', 'm4a'):
        p = os.path.join(base, f'{name}.{ext}')
        if os.path.exists(p):
            return p
    raise FileNotFoundError(f'{scenario}/{name}')


if __name__ == '__main__':
    main()
