"""Generate the editable caption/check icon and Windows sizes with Pillow."""
from pathlib import Path
from PIL import Image, ImageDraw
root = Path(__file__).resolve().parents[1]
scale = 4
im = Image.new('RGBA', (1024, 1024))
d = ImageDraw.Draw(im)
def rect(box, radius, fill):
    d.rounded_rectangle(tuple(v*scale for v in box), radius=radius*scale, fill=fill)
rect((8,8,248,248),48,'#152337')
rect((38,48,218,185),24,'#7CBDF4')
rect((54,64,202,169),12,'#244A71')
rect((72,91,113,105),7,'#EAF6FF')
rect((127,91,181,105),7,'#EAF6FF')
rect((72,124,154,138),7,'#EAF6FF')
d.ellipse((143*scale,143*scale,227*scale,227*scale),fill='#152337')
d.ellipse((151*scale,151*scale,219*scale,219*scale),fill='#F6CF72')
d.line([(166*scale,183*scale),(180*scale,197*scale),(205*scale,170*scale)],fill='#152337',width=10*scale,joint='curve')
im=im.resize((256,256),Image.Resampling.LANCZOS)
im.save(root/'build/appicon.png')
im.save(root/'build/windows/icon.ico',sizes=[(n,n) for n in [16,20,24,32,40,48,64,96,128,256]])
(root/'build/appicon.svg').write_text('''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256">
<rect x="8" y="8" width="240" height="240" rx="48" fill="#152337"/>
<rect x="38" y="48" width="180" height="137" rx="24" fill="#7CBDF4"/>
<rect x="54" y="64" width="148" height="105" rx="12" fill="#244A71"/>
<path d="M79 98h27m28 0h40M79 131h68" stroke="#EAF6FF" stroke-width="14" stroke-linecap="round"/>
<circle cx="185" cy="185" r="42" fill="#152337"/><circle cx="185" cy="185" r="34" fill="#F6CF72"/>
<path d="m166 183 14 14 25-27" stroke="#152337" stroke-width="10" fill="none"/>
</svg>
''',encoding='utf-8')
