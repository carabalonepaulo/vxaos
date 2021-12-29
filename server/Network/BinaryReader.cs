using System;
using System.Text;

namespace Server.Network
{
    public interface IPeekOnlyBinaryReader
    {
        int PeekInt(int index);
        float PeekFloat(int index);
        string PeekString(int index);
        bool PeekBool(int index);
        short PeekShort(int index);
        byte PeekByte(int index);
        byte[] PeekBytes(int index, int length);
    }

    public interface IReadOnlyBinaryReader
    {
        int ReadInt();
        float ReadFloat();
        string ReadString();
        bool ReadBool();
        short ReadShort();
        byte ReadByte();
        byte[] ReadBytes(int length);
    }

    public interface IBinaryReader : IReadOnlyBinaryReader, IPeekOnlyBinaryReader
    {
        int Position { get; }
        byte[] Buffer { get; }
    }

    public class BinaryReader : IBinaryReader
    {
        byte[] buffer;
        int position;

        public byte[] Buffer => buffer;
        public int Position => position;

        public BinaryReader(byte[] buff)
        {
            buffer = buff;
        }

        public byte[] UnreadBuffer
        {
            get
            {
                var buff = new byte[buffer.Length - position];
                Array.Copy(buffer, position, buff, 0, buff.Length);
                return buff;
            }
        }

        public byte[] ReadBytes(int length)
        {
            var value = PeekBytes(position, length);
            position += length;
            return value;
        }

        public bool ReadBool()
        {
            var value = PeekBool(position);
            position += 1;
            return value;
        }

        public byte ReadByte()
        {
            var value = PeekByte(position);
            position += 1;
            return value;
        }

        public short ReadShort()
        {
            var value = PeekShort(position);
            position += 2;
            return value;
        }

        public int ReadInt()
        {
            var value = PeekInt(position);
            position += 4;
            return value;
        }

        public float ReadFloat()
        {
            var value = PeekFloat(position);
            position += 4;
            return value;
        }

        public double ReadDouble()
        {
            var value = PeekDouble(position);
            position += 8;
            return value;
        }

        public string ReadString()
        {
            var value = PeekString(position);
            position += value.Length + 2;
            return value;
        }

        public byte[] PeekBytes(int index, int length)
        {
            var buff = new byte[length];
            Array.Copy(buffer, position, buff, 0, length);
            return buff;
        }

        public byte PeekByte(int index)
        {
            byte value = buffer[position];
            return value;
        }

        public bool PeekBool(int index)
        {
            return PeekByte(index) == 1 ? true : false;
        }

        public short PeekShort(int index)
        {
            var buff = PeekBytes(index, 2);
            short value = BitConverter.ToInt16(buff, 0);
            return value;
        }

        public int PeekInt(int index)
        {
            var buff = new byte[4];
            Array.Copy(buffer, position, buff, 0, 4);
            return BitConverter.ToInt32(buff, 0);
        }

        public float PeekFloat(int index)
        {
            var buff = new byte[4];
            Array.Copy(buffer, position, buff, 0, 4);
            return BitConverter.ToSingle(buff, 0);
        }

        public double PeekDouble(int index)
        {
            var buff = new byte[8];
            Array.Copy(buffer, position, buff, 0, 8);
            return BitConverter.ToDouble(buff, 0);
        }

        public string PeekString(int index)
        {
            var length = PeekShort(index);
            var buff = new byte[length];
            Array.Copy(buffer, position + 2, buff, 0, length);
            return Encoding.UTF8.GetString(buff);
        }
    }
}
