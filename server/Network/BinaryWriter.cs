using System;
using System.Collections.Generic;
using System.Text;
using System.Linq;

namespace Server.Network
{
    public class BinaryWriter
    {
        const int PrefixSize = sizeof(short);
        List<byte[]> _items;

        public BinaryWriter() => _items = new List<byte[]>();

        public void Write(byte[] buff) => _items.Add(buff);

        public void Write(byte value) => _items.Add(new byte[1] { value });

        public void Write(bool value) => Write(value ? (byte)1 : (byte)0);

        public void Write(short value) => Write(BitConverter.GetBytes(value));

        public void Write(int value) => Write(BitConverter.GetBytes(value));

        public void Write(float value) => Write(BitConverter.GetBytes(value));

        public void Write(double value) => Write(BitConverter.GetBytes(value));

        public void Write(string value)
        {
            Write((short)value.Length);
            Write(Encoding.UTF8.GetBytes(value));
        }

        public byte[] ToArray()
        {
            var totalSize = _items.Select(itemBuff => itemBuff.Length).Sum();
            var buff = new byte[totalSize];
            var index = 0;
            foreach (var _itemBuff in _items)
            {
                Array.Copy(_itemBuff, 0, buff, index, _itemBuff.Length);
                index += _itemBuff.Length;
            }
            return buff;
        }

        public byte[] CreatePacketBuffer()
        {
            var totalSize = (short)_items.Select(itemBuff => itemBuff.Length).Sum();
            var buff = new byte[totalSize + PrefixSize];
            var index = PrefixSize;

            Array.Copy(BitConverter.GetBytes(totalSize), 0, buff, 0, PrefixSize);
            foreach (var _itemBuff in _items)
            {
                Array.Copy(_itemBuff, 0, buff, index, _itemBuff.Length);
                index += _itemBuff.Length;
            }
            return buff;
        }
    }
}
