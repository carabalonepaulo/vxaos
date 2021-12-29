using Godot;

namespace API
{
    public class SpriteObject : Node2D
    {
        BitmapObject _bitmap;
        DynamicFont _font;

        public float OriginX = 0;
        public float OriginY = 0;

        public BitmapObject Bitmap
        {
            get { return _bitmap; }
            set
            {
                _bitmap = value;
                this.Update();
            }
        }

        public SpriteObject()
        {
            _font = new DynamicFont();
            _font.FontData = ResourceLoader.Load<DynamicFontData>("res://Fonts/Roboto/Roboto-Regular.ttf");
            _font.Size = 24;
        }

        Color GetColorWithOpacity(byte opacity) => Color.Color8(255, 255, 255, opacity);

        public override void _Draw()
        {
            if (_bitmap == null || _bitmap.Drawables.Count == 0) return;
            foreach (var drawable in _bitmap.Drawables) drawable.Draw(this);
        }
    }

    public class Sprite
    {
        public const int MaxSprites = 9000;

        Bitmap _bitmap;
        ManagedSlots<SpriteObject> _sprites;
        Node _root;

        public Sprite(Bitmap bitmap, Node root)
        {
            _bitmap = bitmap;
            _sprites = new ManagedSlots<SpriteObject>(MaxSprites);
            _root = root;
        }

        public int create()
        {
            var id = _sprites.GetFreeId();
            var sprite = new SpriteObject();
            _root.AddChild(sprite);
            _sprites[id] = sprite;
            return id;
        }

        public void set_color(int spriteId, Color color) => _sprites[spriteId].Modulate = color;

        public void set_bitmap(int spriteId, int bitmapId)
        {
            var sprite = _sprites[spriteId];
            sprite.Bitmap = _bitmap[bitmapId].Clone();
            sprite.Bitmap.NeedRefresh += () => sprite.Update();
        }

        public Vector2 get_position(int spriteId) => _sprites[spriteId].Position;

        public void set_z(int spriteId, int z) => _sprites[spriteId].ZIndex = z;

        public int get_z(int spriteId) => _sprites[spriteId].ZIndex;

        public void set_position(int spriteId, Vector2 position) => _sprites[spriteId].Position = position;

        public void set_visibility(int spriteId, bool visible) => _sprites[spriteId].Visible = visible;

        public bool get_visibility(int spriteId) => _sprites[spriteId].Visible;

        public void set_ox(int spriteId, int ox) => _sprites[spriteId].OriginX = ox;

        public float get_ox(int spriteId) => _sprites[spriteId].OriginX;

        public void set_oy(int spriteId, int oy) => _sprites[spriteId].OriginY = oy;

        public float get_oy(int spriteId) => _sprites[spriteId].OriginY;

        public void dispose(int spriteId)
        {
            var sprite = _sprites[spriteId];
            _root.RemoveChild(sprite);
            sprite.QueueFree();
            _sprites.Free(spriteId);
        }

        public void flash(int spriteId, Color color, int duration) { }

        public void set_zoom(int spriteId, Vector2 zoom) => _sprites[spriteId].Scale = zoom;

        public void set_angle(int spriteId, float angle) => _sprites[spriteId].RotationDegrees = angle;

        public float get_angle(int spriteId) => _sprites[spriteId].RotationDegrees;
    }
}