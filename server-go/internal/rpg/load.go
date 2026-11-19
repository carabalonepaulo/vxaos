package rpg

import (
	"fmt"
	"path/filepath"

	"github.com/kelaia/vxaos/server/internal/rvdata"
)

// Load reads the data files in dir.
func Load(dir string) (*Data, error) {
	d := &Data{
		MapInfos: make(map[int]*MapInfo),
		Maps:     make(map[int]*Map),
	}
	var err error
	if d.Actors, err = loadActors(dir); err != nil {
		return nil, err
	}
	if d.Classes, err = loadClasses(dir); err != nil {
		return nil, err
	}
	if d.Skills, err = loadSkills(dir); err != nil {
		return nil, err
	}
	if d.Items, err = loadItems(dir); err != nil {
		return nil, err
	}
	if d.Weapons, err = loadWeapons(dir); err != nil {
		return nil, err
	}
	if d.Armors, err = loadArmors(dir); err != nil {
		return nil, err
	}
	if d.Enemies, err = loadEnemies(dir); err != nil {
		return nil, err
	}
	if d.States, err = loadStates(dir); err != nil {
		return nil, err
	}
	if d.Animations, err = loadAnimations(dir); err != nil {
		return nil, err
	}
	if d.Tilesets, err = loadTilesets(dir); err != nil {
		return nil, err
	}
	if d.CommonEvents, err = loadCommonEvents(dir); err != nil {
		return nil, err
	}
	if d.System, err = loadSystem(dir); err != nil {
		return nil, err
	}
	if err = d.loadMaps(dir); err != nil {
		return nil, err
	}
	return d, nil
}

func loadRootArray(dir, name string) (*rvdata.Array, error) {
	v, err := rvdata.LoadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	a, ok := v.(*rvdata.Array)
	if !ok {
		return nil, fmt.Errorf("rpg: %s root is %T, want Array", name, v)
	}
	return a, nil
}

func objectAt(a *rvdata.Array, i int) (*rvdata.Object, error) {
	if a == nil || i < 0 || i >= len(a.Elems) {
		return nil, nil
	}
	v := a.Elems[i]
	if v == nil {
		return nil, nil
	}
	o, ok := v.(*rvdata.Object)
	if !ok {
		return nil, fmt.Errorf("rpg: element %d is %T, want Object", i, v)
	}
	return o, nil
}

func field(o *rvdata.Object, name string) any {
	if o == nil {
		return nil
	}
	return o.MustGet(name)
}

func fieldInt(o *rvdata.Object, name string) int {
	n, _ := rvdata.AsInt(field(o, name))
	return int(n)
}

func fieldFloat(o *rvdata.Object, name string) float64 {
	n, _ := rvdata.AsFloat(field(o, name))
	return n
}

func fieldString(o *rvdata.Object, name string) string {
	s, _ := rvdata.AsString(field(o, name))
	return s
}

func fieldBool(o *rvdata.Object, name string) bool {
	b, _ := rvdata.AsBool(field(o, name))
	return b
}

func fieldObject(o *rvdata.Object, name string) *rvdata.Object {
	v, _ := field(o, name).(*rvdata.Object)
	return v
}

func fieldArray(o *rvdata.Object, name string) *rvdata.Array {
	v, _ := field(o, name).(*rvdata.Array)
	return v
}

func fieldTable(o *rvdata.Object, name string) *rvdata.Table {
	v, _ := field(o, name).(*rvdata.Table)
	return v
}

func fieldHash(o *rvdata.Object, name string) *rvdata.Hash {
	v, _ := field(o, name).(*rvdata.Hash)
	return v
}

func ints(o *rvdata.Object, name string) []int {
	a := fieldArray(o, name)
	if a == nil {
		return nil
	}
	out := make([]int, len(a.Elems))
	for i, v := range a.Elems {
		n, _ := rvdata.AsInt(v)
		out[i] = int(n)
	}
	return out
}

func ints8(o *rvdata.Object, name string) [8]int {
	var out [8]int
	values := ints(o, name)
	copy(out[:], values)
	return out
}

func features(o *rvdata.Object) []Feature {
	a := fieldArray(o, "@features")
	if a == nil {
		return nil
	}
	out := make([]Feature, 0, len(a.Elems))
	for _, v := range a.Elems {
		fo, ok := v.(*rvdata.Object)
		if !ok {
			continue
		}
		out = append(out, Feature{
			Code:   fieldInt(fo, "@code"),
			DataID: fieldInt(fo, "@data_id"),
			Value:  fieldFloat(fo, "@value"),
		})
	}
	return out
}

func damage(o *rvdata.Object) Damage {
	do := fieldObject(o, "@damage")
	if do == nil {
		return Damage{}
	}
	return Damage{
		Type:      fieldInt(do, "@type"),
		ElementID: fieldInt(do, "@element_id"),
		Formula:   fieldString(do, "@formula"),
		Variance:  fieldInt(do, "@variance"),
		Critical:  fieldBool(do, "@critical"),
	}
}

func effects(o *rvdata.Object) []Effect {
	a := fieldArray(o, "@effects")
	if a == nil {
		return nil
	}
	out := make([]Effect, 0, len(a.Elems))
	for _, v := range a.Elems {
		eo, ok := v.(*rvdata.Object)
		if !ok {
			continue
		}
		out = append(out, Effect{
			Code:   fieldInt(eo, "@code"),
			DataID: fieldInt(eo, "@data_id"),
			Value1: fieldFloat(eo, "@value1"),
			Value2: fieldFloat(eo, "@value2"),
		})
	}
	return out
}

func loadActors(dir string) ([]*Actor, error) {
	a, err := loadRootArray(dir, "Actors.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Actor, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		out[i] = &Actor{
			ID:             i,
			Name:           fieldString(o, "@name"),
			Description:    fieldString(o, "@description"),
			ClassID:        fieldInt(o, "@class_id"),
			InitialLevel:   fieldInt(o, "@initial_level"),
			MaxLevel:       fieldInt(o, "@max_level"),
			CharacterName:  fieldString(o, "@character_name"),
			CharacterIndex: fieldInt(o, "@character_index"),
			FaceName:       fieldString(o, "@face_name"),
			FaceIndex:      fieldInt(o, "@face_index"),
			Equips:         ints(o, "@equips"),
			Features:       features(o),
			Note:           fieldString(o, "@note"),
		}
	}
	return out, nil
}

func loadClasses(dir string) ([]*Class, error) {
	a, err := loadRootArray(dir, "Classes.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Class, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		exp := ints(o, "@exp_params")
		var expParams [4]int
		copy(expParams[:], exp)
		var learnings []Learning
		if la := fieldArray(o, "@learnings"); la != nil {
			for _, v := range la.Elems {
				lo, ok := v.(*rvdata.Object)
				if !ok {
					continue
				}
				learnings = append(learnings, Learning{
					Level:   fieldInt(lo, "@level"),
					SkillID: fieldInt(lo, "@skill_id"),
				})
			}
		}
		note := fieldString(o, "@note")
		out[i] = &Class{
			ID:        i,
			Name:      fieldString(o, "@name"),
			ExpParams: expParams,
			Params:    fieldTable(o, "@params"),
			Learnings: learnings,
			Features:  features(o),
			Note:      note,
			Graphics:  ParseClassGraphics(note),
		}
	}
	return out, nil
}

func loadUsable(o *rvdata.Object, id int) UsableItem {
	note := fieldString(o, "@note")
	ani, hasAni := NoteNumberPresent(note, "AniIndex")
	u := UsableItem{
		ID:          id,
		Name:        fieldString(o, "@name"),
		Description: fieldString(o, "@description"),
		Note:        note,
		Scope:       fieldInt(o, "@scope"),
		Occasion:    fieldInt(o, "@occasion"),
		HitType:     fieldInt(o, "@hit_type"),
		AnimationID: fieldInt(o, "@animation_id"),
		SuccessRate: fieldInt(o, "@success_rate"),
		Repeats:     fieldInt(o, "@repeats"),
		MPCost:      fieldInt(o, "@mp_cost"),
		TPCost:      fieldInt(o, "@tp_cost"),
		Damage:      damage(o),
		Effects:     effects(o),
		Features:    features(o),
		Range:       NoteNumber(note, "Range"),
		AOE:         NoteNumber(note, "AOE"),
		Level:       NoteNumber(note, "Level"),
		HasAniIndex: hasAni,
		AniIndex:    ani,
		Soulbound:   NoteBool(note, "Soulbound"),
	}
	return u
}

func loadSkills(dir string) ([]*Skill, error) {
	a, err := loadRootArray(dir, "Skills.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Skill, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		out[i] = &Skill{
			UsableItem:       loadUsable(o, i),
			STypeID:          fieldInt(o, "@stype_id"),
			RequiredWTypeID1: fieldInt(o, "@required_wtype_id1"),
			RequiredWTypeID2: fieldInt(o, "@required_wtype_id2"),
		}
	}
	return out, nil
}

func loadItems(dir string) ([]*Item, error) {
	a, err := loadRootArray(dir, "Items.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Item, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		itype := fieldInt(o, "@itype_id")
		out[i] = &Item{
			UsableItem: loadUsable(o, i),
			ITypeID:    itype,
			Price:      fieldInt(o, "@price"),
			Consumable: fieldBool(o, "@consumable"),
			KeyItem:    itype == 2,
		}
	}
	return out, nil
}

func loadWeapons(dir string) ([]*Weapon, error) {
	a, err := loadRootArray(dir, "Weapons.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Weapon, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		note := fieldString(o, "@note")
		ani, hasAni := NoteNumberPresent(note, "AniIndex")
		pdName, pdIndex := ParsePaperdoll(note)
		out[i] = &Weapon{
			ID:             i,
			Name:           fieldString(o, "@name"),
			Description:    fieldString(o, "@description"),
			Note:           note,
			ETypeID:        fieldInt(o, "@etype_id"),
			WTypeID:        fieldInt(o, "@wtype_id"),
			AnimationID:    fieldInt(o, "@animation_id"),
			Params:         ints8(o, "@params"),
			Price:          fieldInt(o, "@price"),
			Features:       features(o),
			Level:          NoteNumber(note, "Level"),
			VIP:            NoteBool(note, "VIP"),
			Soulbound:      NoteBool(note, "Soulbound"),
			HasAniIndex:    hasAni,
			AniIndex:       ani,
			PaperdollName:  pdName,
			PaperdollIndex: pdIndex,
		}
	}
	return out, nil
}

func loadArmors(dir string) ([]*Armor, error) {
	a, err := loadRootArray(dir, "Armors.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Armor, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		note := fieldString(o, "@note")
		pdName, pdIndex := ParsePaperdoll(note)
		etype := fieldInt(o, "@etype_id")
		if t := NoteNumber(note, "Type"); t > 0 {
			etype = t
		}
		sex := 2
		if _, ok := noteValue(note, "Sex"); ok {
			sex = NoteNumber(note, "Sex")
		}
		out[i] = &Armor{
			ID:             i,
			Name:           fieldString(o, "@name"),
			Description:    fieldString(o, "@description"),
			Note:           note,
			ETypeID:        etype,
			ATypeID:        fieldInt(o, "@atype_id"),
			Params:         ints8(o, "@params"),
			Price:          fieldInt(o, "@price"),
			Features:       features(o),
			Level:          NoteNumber(note, "Level"),
			VIP:            NoteBool(note, "VIP"),
			Soulbound:      NoteBool(note, "Soulbound"),
			Sex:            sex,
			PaperdollName:  pdName,
			PaperdollIndex: pdIndex,
		}
	}
	return out, nil
}

func loadEnemies(dir string) ([]*Enemy, error) {
	a, err := loadRootArray(dir, "Enemies.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Enemy, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		note := fieldString(o, "@note")
		ani, hasAni := NoteNumberPresent(note, "AniIndex")
		var drops []EnemyDrop
		if da := fieldArray(o, "@drop_items"); da != nil {
			for _, v := range da.Elems {
				do, ok := v.(*rvdata.Object)
				if !ok {
					continue
				}
				drops = append(drops, EnemyDrop{
					Kind:        fieldInt(do, "@kind"),
					DataID:      fieldInt(do, "@data_id"),
					Denominator: fieldInt(do, "@denominator"),
				})
			}
		}
		var actions []EnemyAction
		if aa := fieldArray(o, "@actions"); aa != nil {
			for _, v := range aa.Elems {
				ao, ok := v.(*rvdata.Object)
				if !ok {
					continue
				}
				actions = append(actions, EnemyAction{
					SkillID:         fieldInt(ao, "@skill_id"),
					ConditionType:   fieldInt(ao, "@condition_type"),
					ConditionParam1: fieldInt(ao, "@condition_param1"),
					ConditionParam2: fieldInt(ao, "@condition_param2"),
					Rating:          fieldInt(ao, "@rating"),
				})
			}
		}
		out[i] = &Enemy{
			ID:                i,
			Name:              fieldString(o, "@name"),
			Description:       fieldString(o, "@description"),
			Params:            ints8(o, "@params"),
			Exp:               fieldInt(o, "@exp"),
			Gold:              fieldInt(o, "@gold"),
			DropItems:         drops,
			Actions:           actions,
			Features:          features(o),
			Note:              note,
			Sight:             NoteNumber(note, "Sight"),
			ReviveTime:        NoteNumber(note, "ReviveTime"),
			DisableSwitchID:   NoteNumber(note, "SwitchID"),
			DisableVariableID: NoteNumber(note, "VariableID"),
			HasAniIndex:       hasAni,
			AniIndex:          ani,
			Escape:            NoteBool(note, "Escape"),
			Boss:              NoteBool(note, "Boss"),
		}
	}
	return out, nil
}

func loadStates(dir string) ([]*State, error) {
	a, err := loadRootArray(dir, "States.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*State, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		out[i] = &State{
			ID:                  i,
			Name:                fieldString(o, "@name"),
			IconIndex:           fieldInt(o, "@icon_index"),
			Restriction:         fieldInt(o, "@restriction"),
			Priority:            fieldInt(o, "@priority"),
			RemoveByRestriction: fieldBool(o, "@remove_by_restriction"),
			RemoveByDamage:      fieldBool(o, "@remove_by_damage"),
			ChanceByDamage:      fieldInt(o, "@chance_by_damage"),
			AutoRemovalTiming:   fieldInt(o, "@auto_removal_timing"),
			MinTurns:            fieldInt(o, "@min_turns"),
			MaxTurns:            fieldInt(o, "@max_turns"),
			StepsToRemove:       fieldInt(o, "@steps_to_remove"),
			Features:            features(o),
			Note:                fieldString(o, "@note"),
		}
	}
	return out, nil
}

func loadAnimations(dir string) ([]*Animation, error) {
	a, err := loadRootArray(dir, "Animations.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Animation, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		out[i] = &Animation{
			ID:       i,
			Name:     fieldString(o, "@name"),
			FrameMax: fieldInt(o, "@frame_max"),
		}
	}
	return out, nil
}

func loadTilesets(dir string) ([]*Tileset, error) {
	a, err := loadRootArray(dir, "Tilesets.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*Tileset, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		out[i] = &Tileset{
			ID:    i,
			Name:  fieldString(o, "@name"),
			Mode:  fieldInt(o, "@mode"),
			Flags: fieldTable(o, "@flags"),
			Note:  fieldString(o, "@note"),
		}
	}
	return out, nil
}

func loadCommonEvents(dir string) ([]*CommonEvent, error) {
	a, err := loadRootArray(dir, "CommonEvents.rvdata2")
	if err != nil {
		return nil, err
	}
	out := make([]*CommonEvent, len(a.Elems))
	for i := 0; i < len(a.Elems); i++ {
		o, err := objectAt(a, i)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}
		list, err := commands(fieldArray(o, "@list"))
		if err != nil {
			return nil, err
		}
		out[i] = &CommonEvent{
			ID:       i,
			Name:     fieldString(o, "@name"),
			Trigger:  fieldInt(o, "@trigger"),
			SwitchID: fieldInt(o, "@switch_id"),
			List:     list,
		}
	}
	return out, nil
}

func loadSystem(dir string) (*System, error) {
	v, err := rvdata.LoadFile(filepath.Join(dir, "System.rvdata2"))
	if err != nil {
		return nil, err
	}
	o, ok := v.(*rvdata.Object)
	if !ok {
		return nil, fmt.Errorf("rpg: System root is %T", v)
	}
	return &System{
		GameTitle:     fieldString(o, "@game_title"),
		StartMapID:    fieldInt(o, "@start_map_id"),
		StartX:        fieldInt(o, "@start_x"),
		StartY:        fieldInt(o, "@start_y"),
		OptFloorDeath: fieldBool(o, "@opt_floor_death"),
		CurrencyUnit:  fieldString(o, "@currency_unit"),
	}, nil
}

func (d *Data) loadMaps(dir string) error {
	v, err := rvdata.LoadFile(filepath.Join(dir, "MapInfos.rvdata2"))
	if err != nil {
		return err
	}
	h, ok := v.(*rvdata.Hash)
	if !ok {
		return fmt.Errorf("rpg: MapInfos root is %T", v)
	}
	for _, pair := range h.Pairs {
		id, ok := rvdata.AsInt(pair.Key)
		if !ok || id <= 0 {
			continue
		}
		o, ok := pair.Value.(*rvdata.Object)
		if !ok {
			continue
		}
		d.MapInfos[int(id)] = &MapInfo{
			ID:       int(id),
			Name:     fieldString(o, "@name"),
			ParentID: fieldInt(o, "@parent_id"),
			Order:    fieldInt(o, "@order"),
		}
	}
	for id, info := range d.MapInfos {
		m, err := loadMap(dir, id, info.Name)
		if err != nil {
			return err
		}
		d.Maps[id] = m
	}
	return nil
}

func loadMap(dir string, id int, name string) (*Map, error) {
	v, err := rvdata.LoadFile(filepath.Join(dir, fmt.Sprintf("Map%03d.rvdata2", id)))
	if err != nil {
		return nil, err
	}
	o, ok := v.(*rvdata.Object)
	if !ok {
		return nil, fmt.Errorf("rpg: Map%03d root is %T", id, v)
	}
	note := fieldString(o, "@note")
	m := &Map{
		ID:        id,
		Name:      name,
		Width:     fieldInt(o, "@width"),
		Height:    fieldInt(o, "@height"),
		TilesetID: fieldInt(o, "@tileset_id"),
		Note:      note,
		Data:      fieldTable(o, "@data"),
		Events:    make(map[int]*Event),
		PvP:       NoteBool(note, "PvP"),
	}
	if m.Width <= 0 || m.Height <= 0 {
		return nil, fmt.Errorf("rpg: Map%03d has invalid dimensions %dx%d", id, m.Width, m.Height)
	}
	events := fieldHash(o, "@events")
	if events != nil {
		for _, pair := range events.Pairs {
			eventID, ok := rvdata.AsInt(pair.Key)
			if !ok || eventID <= 0 {
				continue
			}
			eo, ok := pair.Value.(*rvdata.Object)
			if !ok {
				continue
			}
			ev, err := loadEvent(eo, int(eventID))
			if err != nil {
				return nil, fmt.Errorf("rpg: Map%03d event %d: %w", id, eventID, err)
			}
			m.Events[int(eventID)] = ev
		}
	}
	return m, nil
}

func loadEvent(o *rvdata.Object, id int) (*Event, error) {
	ev := &Event{
		ID:   id,
		Name: fieldString(o, "@name"),
		X:    fieldInt(o, "@x"),
		Y:    fieldInt(o, "@y"),
	}
	pages := fieldArray(o, "@pages")
	if pages == nil {
		return ev, nil
	}
	for _, v := range pages.Elems {
		po, ok := v.(*rvdata.Object)
		if !ok {
			continue
		}
		page, err := loadPage(po)
		if err != nil {
			return nil, err
		}
		ev.Pages = append(ev.Pages, page)
	}
	return ev, nil
}

func loadPage(o *rvdata.Object) (*Page, error) {
	p := &Page{
		MoveType:      fieldInt(o, "@move_type"),
		MoveSpeed:     fieldInt(o, "@move_speed"),
		MoveFrequency: fieldInt(o, "@move_frequency"),
		WalkAnime:     fieldBool(o, "@walk_anime"),
		StepAnime:     fieldBool(o, "@step_anime"),
		DirectionFix:  fieldBool(o, "@direction_fix"),
		Through:       fieldBool(o, "@through"),
		PriorityType:  fieldInt(o, "@priority_type"),
		Trigger:       fieldInt(o, "@trigger"),
	}
	if c := fieldObject(o, "@condition"); c != nil {
		p.Condition = Condition{
			Switch1Valid:    fieldBool(c, "@switch1_valid"),
			Switch1ID:       fieldInt(c, "@switch1_id"),
			Switch2Valid:    fieldBool(c, "@switch2_valid"),
			Switch2ID:       fieldInt(c, "@switch2_id"),
			VariableValid:   fieldBool(c, "@variable_valid"),
			VariableID:      fieldInt(c, "@variable_id"),
			VariableValue:   fieldInt(c, "@variable_value"),
			SelfSwitchValid: fieldBool(c, "@self_switch_valid"),
			SelfSwitchCh:    fieldString(c, "@self_switch_ch"),
			ItemValid:       fieldBool(c, "@item_valid"),
			ItemID:          fieldInt(c, "@item_id"),
			ActorValid:      fieldBool(c, "@actor_valid"),
			ActorID:         fieldInt(c, "@actor_id"),
		}
	}
	if g := fieldObject(o, "@graphic"); g != nil {
		p.Graphic = Graphic{
			TileID:         fieldInt(g, "@tile_id"),
			CharacterName:  fieldString(g, "@character_name"),
			CharacterIndex: fieldInt(g, "@character_index"),
			Direction:      fieldInt(g, "@direction"),
			Pattern:        fieldInt(g, "@pattern"),
		}
	}
	if r := fieldObject(o, "@move_route"); r != nil {
		mr := &MoveRoute{
			Repeat:    fieldBool(r, "@repeat"),
			Skippable: fieldBool(r, "@skippable"),
			Wait:      fieldBool(r, "@wait"),
		}
		if list := fieldArray(r, "@list"); list != nil {
			for _, v := range list.Elems {
				co, ok := v.(*rvdata.Object)
				if !ok {
					continue
				}
				mr.List = append(mr.List, MoveCommand{
					Code:       fieldInt(co, "@code"),
					Parameters: parameters(fieldArray(co, "@parameters")),
				})
			}
		}
		p.MoveRoute = mr
	}
	list, err := commands(fieldArray(o, "@list"))
	if err != nil {
		return nil, err
	}
	p.List = list
	return p, nil
}

func commands(a *rvdata.Array) ([]*EventCommand, error) {
	if a == nil {
		return nil, nil
	}
	out := make([]*EventCommand, 0, len(a.Elems))
	for _, v := range a.Elems {
		o, ok := v.(*rvdata.Object)
		if !ok {
			continue
		}
		out = append(out, &EventCommand{
			Code:       fieldInt(o, "@code"),
			Indent:     fieldInt(o, "@indent"),
			Parameters: parameters(fieldArray(o, "@parameters")),
		})
	}
	return out, nil
}

func parameters(a *rvdata.Array) []any {
	if a == nil {
		return nil
	}
	out := make([]any, len(a.Elems))
	copy(out, a.Elems)
	return out
}
