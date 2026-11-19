#==============================================================================
# ** Test Input
#------------------------------------------------------------------------------
#  Test-only cursor/button override. VXAOS.dll exposes the injected client
#  position and merges injected buttons into Input; this script routes the
#  engine's Mouse.x/Mouse.y/click?/press? through that state. Disabled unless
#  the process was started with VXAOS_TEST_INPUT (the input-bridge switch).
#==============================================================================

if ENV['VXAOS_TEST_INPUT']
  begin
    module Mouse
      class << self
        alias_method :vxaos_native_x, :x unless method_defined?(:vxaos_native_x)
        alias_method :vxaos_native_y, :y unless method_defined?(:vxaos_native_y)
        alias_method :vxaos_native_click, :click? unless method_defined?(:vxaos_native_click)
        alias_method :vxaos_native_press, :press? unless method_defined?(:vxaos_native_press)
        if method_defined?(:dbl_clk?)
          alias_method :vxaos_native_dbl_clk, :dbl_clk? unless method_defined?(:vxaos_native_dbl_clk)
        end
        VXAOS_MOUSE_POS = c_function('System/VXAOS.dll',
          'int VxaosInputTest__mousePos(long*)') unless const_defined?(:VXAOS_MOUSE_POS)

        def vxaos_test_position
          @vxaos_test_position ||= DL::CPtr.new(DL.malloc(8), 8)
          return nil unless VXAOS_MOUSE_POS.call(@vxaos_test_position) == 1
          @vxaos_test_position.to_str.unpack('l2')
        end

        def x
          position = vxaos_test_position
          position ? position[0] : vxaos_native_x
        end

        def y
          position = vxaos_test_position
          position ? position[1] : vxaos_native_y
        end

        def click?(button)
          case button
          when :L then Input.trigger?(:LBUTTON)
          when :R then Input.trigger?(:RBUTTON)
          else vxaos_native_click(button)
          end
        end

        def press?(button)
          case button
          when :L then Input.press?(:LBUTTON)
          when :R then Input.press?(:RBUTTON)
          else vxaos_native_press(button)
          end
        end

        # The injected buttons are single-frame presses, so emulate the
        # platform double-click detector: two presses of the same button within
        # 350 ms.
        def dbl_clk?(button)
          unless button == :L || button == :R
            return vxaos_native_dbl_clk(button) if respond_to?(:vxaos_native_dbl_clk)
            return false
          end
          key = button == :L ? :LBUTTON : :RBUTTON
          return false unless Input.trigger?(key)
          now = Time.now
          if @vxaos_test_click_at && now - @vxaos_test_click_at <= 0.35
            @vxaos_test_click_at = nil
            return true
          end
          @vxaos_test_click_at = now
          false
        end
      end
    end
  rescue => error
    begin
      File.open('System/input-test.log', 'a') do |file|
        file.puts("#{Time.now}: test input failed: #{error.class}: #{error.message}")
      end
    rescue
    end
  end

  begin
    class Window_ChoiceList
      alias_method :vxaos_choice_start, :start unless method_defined?(:vxaos_choice_start)
      def start
        vxaos_choice_start
        File.open('System/input-test.log', 'a') do |file|
          file.puts("#{Time.now}: choice start x=#{x} y=#{y} w=#{width} h=#{height} index=#{index}")
        end
      rescue
      end

      alias_method :vxaos_choice_update, :update unless method_defined?(:vxaos_choice_update)
      def update
        vxaos_choice_update
        if Input.trigger?(:LBUTTON)
          File.open('System/input-test.log', 'a') do |file|
            file.puts("#{Time.now}: choice click index=#{index} mouse=#{Mouse.x},#{Mouse.y} active=#{active} open=#{open?}")
          end
        end
      rescue
      end
    end
  rescue
  end

  begin
    class Window_Message
      alias_method :vxaos_native_update, :update unless method_defined?(:vxaos_native_update)
      def update
        vxaos_native_update
        text = $game_message.has_text?
        if @vxaos_last_text != text
          @vxaos_last_text = text
          begin
            File.open('System/input-test.log', 'a') { |file| file.puts("#{Time.now}: message text=#{text}") }
          rescue
          end
        end
      end
    end
  rescue
  end
end
