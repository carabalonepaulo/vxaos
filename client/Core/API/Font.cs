using Godot;
using System;
using System.Collections.Generic;

namespace API
{
    public class FontObject
    {
        static string[] Variants = new string[] { "Bold", "Italic", "BoldItalic", "Regular" };

        DynamicFont _font;
        Color _color;
        Color _out_color;
        int _out_size;
        bool _bold;
        bool _italic;
        bool _shadow;
        // Dictionary<string, DynamicFontData> _variants;
        Dictionary<string, DynamicFont> _variants;

        public DynamicFont DynamicFont { get { return _font; } }
        public int Height
        {
            get { return _font.Size; }
            set { foreach (var variant in Variants) { _variants[variant].Size = value; } }
        }
        public Color Color
        {
            get { return _color; }
            set { _color = value; }
        }
        public Color OutColor
        {
            get { return _out_color; }
            set { _out_color = value; }
        }
        public int OutSize
        {
            get { return _out_size; }
            set { _out_size = value; }
        }
        public bool Bold
        {
            get { return _bold; }
            set { _bold = value; Update(); }
        }
        public bool Italic
        {
            get { return _italic; }
            set { _italic = value; Update(); }
        }
        public bool Outline
        {
            get { return _font.HasOutline(); }
            set
            {
                _font.OutlineSize = value ? 1 : 0;
                _font.OutlineColor = value ? _out_color : Colors.Transparent;
            }
        }
        public bool Shadow
        {
            get { return _shadow; }
            set { _shadow = value; }
        }

        string GetFontPath(string fontName, string variant)
        {
            var godotFS = $"res://Fonts/{fontName}";
            var hostFS = $"./Fonts/{fontName}";
            var basePath = (Main.LocalFileSystem ? hostFS : godotFS);
            return $"{basePath}/{fontName}-{variant}.ttf";
        }

        Dictionary<string, DynamicFont> LoadVariants(string fontName)
        {
            var variants = new Dictionary<string, DynamicFont>();
            foreach (var variant in Variants)
            {
                var path = GetFontPath(fontName, variant);
                var font = new DynamicFont();
                font.FontData = ResourceLoader.Load<DynamicFontData>(path);
                variants.Add(variant, font);
            }
            return variants;
        }

        public FontObject(string name, int height)
        {
            _variants = LoadVariants(name);
            _font = _variants["Regular"];
        }

        void Update()
        {
            if (_bold && _italic) { _font = _variants["BoldItalic"]; }
            else if (_bold) { _font = _variants["Bold"]; }
            else if (_italic) { _font = _variants["Italic"]; }
            else { _font = _variants["Regular"]; }
        }

        public void Dispose()
        {
            foreach (var variant in Variants) { _variants[variant].Dispose(); }
        }
    }

    public class Font
    {
        ManagedSlots<FontObject> _fonts;

        public Font()
        {
            _fonts = new ManagedSlots<FontObject>(256);
        }

        public FontObject this[int fontId]
        {
            get { return _fonts[fontId]; }
            private set { _fonts[fontId] = value; }
        }

        public int create(string fontName, int size)
        {
            var id = _fonts.GetFreeId();
            _fonts[id] = new FontObject(fontName, size);
            return id;
        }

        public void dispose(int fontId)
        {
            _fonts[fontId].Dispose();
            _fonts.Free(fontId);
        }

        public void set_size(int fontId, int size) { }

        public void set_bold(int fontId, bool value) { _fonts[fontId].Bold = value; }

        public void set_italic(int fontId, bool value) { _fonts[fontId].Italic = value; }

        public void set_outline(int fontId, bool value) { _fonts[fontId].Outline = value; }

        public void set_color(int fontId, Color color) { _fonts[fontId].Color = color; }

        public void set_out_color(int fontId, Color color) { _fonts[fontId].OutColor = color; }

        public void set_out_size(int fontId, int size) { _fonts[fontId].OutSize = size; }

        public void set_shadow(int fontId, bool value) { _fonts[fontId].Shadow = value; }
    }
}