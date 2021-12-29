using System;
using Godot;
using Microsoft.Scripting.Hosting;

public class Main : Node2D
{
    public static bool LocalFileSystem = false;

    System.Threading.Thread _thread;
    AcceptDialog _dialog;
    ColorRect _panel;
    RichTextLabel _statusLabel;
    dynamic _main;

    ScriptEngine _engine;
    ScriptRuntime _runtime;
    ScriptScope _scope;
    bool _hasError = false;
    API.System _system;
    API.Bitmap _bitmap;
    API.Sprite _sprite;
    API.Font _font;

    public override void _Ready()
    {
        _dialog = GetNode<AcceptDialog>("AcceptDialog");
        _dialog.Connect("confirmed", this, nameof(OnDialogConfirmed));
        _dialog.WindowTitle = "Error";

        _panel = GetNode<ColorRect>("ColorRect");
        _statusLabel = GetNode<RichTextLabel>("ColorRect/RichTextLabel");

        _system = new API.System(this, GetCanvasItem(), GetViewport(), _statusLabel, new API.SignalHandler(_engine));
        _system.Error += PopupError;

        _font = new API.Font();
        _bitmap = new API.Bitmap(_font);
        _sprite = new API.Sprite(_bitmap, this);

        CreateRubyEngine();
    }

    public override void _Process(float delta)
    {
        if (_main == null || _hasError) return;
        try { _engine.Operations.InvokeMember(_main, "update", delta); }
        catch (IronRuby.Builtins.RuntimeError err) { PopupError(err.Message); }
    }

    public override void _Input(InputEvent inputEvent)
    {
        if (inputEvent is InputEventKey keyEvent && keyEvent.Pressed)
        {
            if ((KeyList)keyEvent.Scancode == KeyList.F2)
            {
                _statusLabel.Visible = !_statusLabel.Visible;
                _panel.Visible = _statusLabel.Visible;
            }
        }
    }

    void PopupError(string text)
    {
        _hasError = true;
        _dialog.DialogText = text;
        _dialog.PopupCentered();
    }

    void OnDialogConfirmed()
    {
        GetTree().Quit();
    }

    public override void _Notification(int what)
    {
        if (_main == null) return;
        _engine.Operations.InvokeMember(_main, "notification", what);
    }

    string RubyMain => @"
load_assembly 'IronRuby.Libraries', 'IronRuby.StandardLibrary.Zlib'

module API
  System = nil
  Primitives = nil
  FileSystem = nil
  Bitmap = nil
  Sprite = nil
  Font = nil

  def self.initialize(system, primitives, file_system, bitmap, sprite, font)
    API::System = system
    API::Primitives = primitives
    API::FileSystem = file_system
    API::Bitmap = bitmap
    API::Sprite = sprite
    API::Font = font

    API::System.log 'red', 'ruby', 'Loading RPG Maker scripts...'

    scripts_file_path = 'res://Client/Data/Scripts.rvdata2'
    #scripts_file_path = './Data/Scripts.rvdata2'
    if API::FileSystem.exists scripts_file_path
      buff = API::FileSystem.read_all_bytes scripts_file_path
      Marshal.load(buff).each do |script|
        begin
          eval(Zlib::Inflate.inflate(script[2]).force_encoding('UTF-8'))
        rescue => e
          API::System.raise e.inspect
        end
      end
      API::System.log 'red', 'ruby', 'Scripts successfully loaded.'
    else
      API::System.log 'red', 'ruby', 'Scripts.rvdata2 not found.'
      API::System.raise 'Scripts.rvdata2 not found.'
    end
  end
end
";

    void WriteTestFile()
    {
        var file = new File();
        file.Open("res://test.txt", File.ModeFlags.Write);
        file.StoreString("test");
        file.Close();
    }

    void CreateRubyEngine()
    {
        _system.log("#42f54b", "debug", "Initializing ruby vm.");

        _runtime = IronRuby.Ruby.CreateRuntime();
        _engine = _runtime.GetEngine("Ruby");
        _scope = _engine.CreateScope();

        _engine.Execute(RubyMain, _scope);
        var api = _engine.Runtime.Globals.GetVariable("API");
        _engine.Operations.InvokeMember(api, "initialize", _system, new API.Primitives(),
            new API.FileSystem(), _bitmap, _sprite, _font);

        try
        {
            var main = _engine.Runtime.Globals.GetVariable("Main");
            _main = _engine.Operations.CreateInstance(main);
        }
        catch (Exception e) { _system.raise(e.Message); }
    }
}
