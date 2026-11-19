#==============================================================================
# ** Server Authority
#------------------------------------------------------------------------------
#  O servidor é a única autoridade de dano/fórmula. O client só exibe valores
#  cosméticos; qualquer tentativa de avaliar a fórmula localmente retorna 0.
#==============================================================================

class RPG::UsableItem::Damage
  def eval(a, b, v)
    0
  end
end
