#==============================================================================
# ** Game_Guild
#------------------------------------------------------------------------------
#  Estado da guilda no cliente, preenchido pelas mensagens do servidor.
#==============================================================================

class Game_Guild

  attr_accessor :leader, :notice, :flag, :members, :online_size

  def initialize
    @leader = ''
    @notice = ''
    @flag = Array.new(64, 255)
    @members = []
    @online_size = 0
  end

end

$game_guild = Game_Guild.new
