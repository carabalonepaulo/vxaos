using Godot;
using System;
using System.Collections.Generic;
using System.Linq;
using RGSSPlayer;
using RGSSPlayer.Drawables;

namespace API
{
    public class BitmapObject
    {
        public event Action NeedRefresh;

        Texture _texture;
        private List<Drawable> _drawables;
        Vector2 _size;
        FontObject _font;

        public int Id;
        public List<Drawable> Drawables { get { return _drawables; } }
        public FontObject Font { get { return _font; } set { _font = value; } }

        public Texture Texture
        {
            get { return _texture; }
            set
            {
                _texture = value;
                _size = _texture.GetSize();
            }
        }
        public bool IsUsingTexture => Texture != null;

        public Vector2 Size
        {
            get { return _size; }
            set
            {
                _size = value;
            }
        }

        public BitmapObject()
        {
            _drawables = new List<Drawable>();
        }

        public BitmapObject(List<Drawable> drawCalls)
        {
            _drawables = new List<Drawable>(drawCalls);
        }

        public void AddDrawCall(Drawable drawable)
        {
            _drawables.Add(drawable);
            NeedRefresh?.Invoke();
        }

        public void ClearCalls()
        {
            _drawables.Clear();
            NeedRefresh?.Invoke();
        }

        public void CopyFrom(List<Drawable> drawCalls)
        {
            _drawables = new List<Drawable>(drawCalls);
        }

        public BitmapObject Clone() => new BitmapObject(_drawables);
    }

    public class Bitmap
    {
        public const int MaxBitmaps = 9000;
        ManagedSlots<BitmapObject> _bitmaps;
        ITextureLoader _textureLoader;
        API.Font _fontAPI;

        public Bitmap(API.Font fontAPI)
        {
            _fontAPI = fontAPI;
            var factory = new TextureLoaderFactory();
            _textureLoader = factory.CreateTextureLoader();
            _bitmaps = new ManagedSlots<BitmapObject>(MaxBitmaps);
        }

        public bool intersect(int x, int y, Rect2 rect) => x >= rect.Position.x && x <= rect.Position.x + rect.Size.x && y >= rect.Position.y && y <= rect.Position.y + rect.Size.y;

        public BitmapObject this[int bitmapId]
        {
            get { return _bitmaps[bitmapId]; }
            private set { _bitmaps[bitmapId] = value; }
        }

        public int create(int width, int height)
        {
            var bitmap = new BitmapObject();
            bitmap.Size = new Vector2(width, height);

            var id = _bitmaps.GetFreeId();
            _bitmaps[id] = bitmap;
            return id;
        }

        public int create(string path)
        {
            var bitmap = new BitmapObject();
            bitmap.Texture = _textureLoader.Load(path);

            var id = _bitmaps.GetFreeId();
            _bitmaps[id] = bitmap;
            return id;
        }

        public void dispose(int bitmapId)
        {
            _bitmaps.Free(bitmapId);
        }

        public Vector2 get_size(int bitmapId) => _bitmaps[bitmapId].Size;

        public float get_width(int bitmapId) => _bitmaps[bitmapId].Size.x;

        public float get_height(int bitmapId) => _bitmaps[bitmapId].Size.y;

        public void set_font(int bitmapId, int fontId) => _bitmaps[bitmapId].Font = _fontAPI[fontId];

        public void clear(int bitmapId)
        {
            _bitmaps[bitmapId].ClearCalls();
        }

        public void draw_text(int bitmapId, Rect2 rect, string text, int align = 0)
        {
            var font = _bitmaps[bitmapId].Font;
            var dynamicFont = _bitmaps[bitmapId].Font.DynamicFont;
            var drawable = new DrawString(rect, text, align, dynamicFont, font.Color);
            _bitmaps[bitmapId].AddDrawCall(drawable);
        }

        public void fill_rect(int bitmapId, Rect2 rect, Color color)
        {
            _bitmaps[bitmapId].AddDrawCall(new DrawRect(rect, color));
        }

        public void blt(int bitmapId, int x, int y, int sourceBitmapId, Rect2 sourceRect, byte opacity = 255)
        {
            var destRect = new Rect2(x, y, sourceRect.Size.x, sourceRect.Size.y);
            stretch_blt(bitmapId, destRect, sourceBitmapId, sourceRect, opacity);
        }

        public void stretch_blt(int bitmapId, Rect2 destRect, int sourceBitmapId, Rect2 sourceRect, byte opacity = 255)
        {
            var texture = _bitmaps[sourceBitmapId].Texture;
            _bitmaps[bitmapId].AddDrawCall(new DrawTextureRectRegion(destRect, texture, sourceRect, opacity));
        }

        public Color get_pixel(int bitmapId, int x, int y)
        {
            var bitmap = _bitmaps[bitmapId];
            var calls = bitmap.Drawables.ToArray();
            Color? finalColor = null;

            for (var i = calls.Length - 1; i >= 0; i--)
            {
                var drawbale = calls[i];
                if (drawbale.Intersect(x, y))
                {
                    finalColor = drawbale.GetPixel(x, y);
                    break;
                }
            }

            return finalColor == null ? Colors.White : (Color)finalColor;
        }

        public void set_pixel(int bitmapId, int x, int y, Color color)
        {
            _bitmaps[bitmapId].AddDrawCall(new DrawRect(new Rect2(x, y, 1, 1), color));
        }
    }
}