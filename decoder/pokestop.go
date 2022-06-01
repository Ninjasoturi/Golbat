package decoder

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/jellydator/ttlcache/v3"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"golbat/pogo"
	"golbat/util"
	"golbat/webhooks"
	"gopkg.in/guregu/null.v4"
	"strings"
	"time"
)

type Pokestop struct {
	Id                         string      `db:"id"`
	Lat                        float64     `db:"lat"`
	Lon                        float64     `db:"lon"`
	Name                       null.String `db:"name"`
	Url                        null.String `db:"url"`
	LureExpireTimestamp        null.Int    `db:"lure_expire_timestamp"`
	LastModifiedTimestamp      null.Int    `db:"last_modified_timestamp"`
	Updated                    int64       `db:"updated"`
	Enabled                    null.Bool   `db:"enabled"`
	QuestType                  null.Int    `db:"quest_type"`
	QuestTimestamp             null.Int    `db:"quest_timestamp"`
	QuestTarget                null.Int    `db:"quest_target"`
	QuestConditions            null.String `db:"quest_conditions"`
	QuestRewards               null.String `db:"quest_rewards"`
	QuestTemplate              null.String `db:"quest_template"`
	QuestTitle                 null.String `db:"quest_title"`
	CellId                     null.Int    `db:"cell_id"`
	Deleted                    bool        `db:"deleted"`
	LureId                     int16       `db:"lure_id"`
	FirstSeenTimestamp         int16       `db:"first_seen_timestamp"`
	SponsorId                  null.Int    `db:"sponsor_id"`
	PartnerId                  null.String `db:"partner_id"`
	ArScanEligible             null.Int    `db:"ar_scan_eligible"` // is an 8
	PowerUpLevel               null.Int    `db:"power_up_level"`
	PowerUpPoints              null.Int    `db:"power_up_points"`
	PowerUpEndTimestamp        null.Int    `db:"power_up_end_timestamp"`
	AlternativeQuestType       null.Int    `db:"alternative_quest_type"`
	AlternativeQuestTimestamp  null.Int    `db:"alternative_quest_timestamp"`
	AlternativeQuestTarget     null.Int    `db:"alternative_quest_target"`
	AlternativeQuestConditions null.String `db:"alternative_quest_conditions"`
	AlternativeQuestRewards    null.String `db:"alternative_quest_rewards"`
	AlternativeQuestTemplate   null.String `db:"alternative_quest_template"`
	AlternativeQuestTitle      null.String `db:"alternative_quest_title"`

	//`id` varchar(35) NOT NULL,
	//`lat` double(18,14) NOT NULL,
	//`lon` double(18,14) NOT NULL,
	//`name` varchar(128) DEFAULT NULL,
	//`url` varchar(200) DEFAULT NULL,
	//`lure_expire_timestamp` int unsigned DEFAULT NULL,
	//`last_modified_timestamp` int unsigned DEFAULT NULL,
	//`updated` int unsigned NOT NULL,
	//`enabled` tinyint unsigned DEFAULT NULL,
	//`quest_type` int unsigned DEFAULT NULL,
	//`quest_timestamp` int unsigned DEFAULT NULL,
	//`quest_target` smallint unsigned DEFAULT NULL,
	//`quest_conditions` text,
	//`quest_rewards` text,
	//`quest_template` varchar(100) DEFAULT NULL,
	//`quest_title` varchar(100) DEFAULT NULL,
	//`quest_reward_type` smallint unsigned GENERATED ALWAYS AS (json_extract(json_extract(`quest_rewards`,_utf8mb4'$[*].type'),_utf8mb4'$[0]')) VIRTUAL,
	//`quest_item_id` smallint unsigned GENERATED ALWAYS AS (json_extract(json_extract(`quest_rewards`,_utf8mb4'$[*].info.item_id'),_utf8mb4'$[0]')) VIRTUAL,
	//`quest_reward_amount` smallint unsigned GENERATED ALWAYS AS (json_extract(json_extract(`quest_rewards`,_utf8mb4'$[*].info.amount'),_utf8mb4'$[0]')) VIRTUAL,
	//`cell_id` bigint unsigned DEFAULT NULL,
	//`deleted` tinyint unsigned NOT NULL DEFAULT '0',
	//`lure_id` smallint DEFAULT '0',
	//`first_seen_timestamp` int unsigned NOT NULL,
	//`sponsor_id` smallint unsigned DEFAULT NULL,
	//`partner_id` varchar(35) DEFAULT NULL,
	//`quest_pokemon_id` smallint unsigned GENERATED ALWAYS AS (json_extract(json_extract(`quest_rewards`,_utf8mb4'$[*].info.pokemon_id'),_utf8mb4'$[0]')) VIRTUAL,
	//`ar_scan_eligible` tinyint unsigned DEFAULT NULL,
	//`power_up_level` smallint unsigned DEFAULT NULL,
	//`power_up_points` int unsigned DEFAULT NULL,
	//`power_up_end_timestamp` int unsigned DEFAULT NULL,
	//`alternative_quest_type` int unsigned DEFAULT NULL,
	//`alternative_quest_timestamp` int unsigned DEFAULT NULL,
	//`alternative_quest_target` smallint unsigned DEFAULT NULL,
	//`alternative_quest_conditions` text,
	//`alternative_quest_rewards` text,
	//`alternative_quest_template` varchar(100) DEFAULT NULL,
	//`alternative_quest_title` varchar(100) DEFAULT NULL,

}

func getPokestopRecord(db *sqlx.DB, fortId string) (*Pokestop, error) {
	stop := pokestopCache.Get(fortId)
	if stop != nil {
		pokestop := stop.Value()
		return &pokestop, nil
	}
	pokestop := Pokestop{}
	err := db.Get(&pokestop,
		"SELECT pokestop.id, lat, lon, name, url, enabled, lure_expire_timestamp, last_modified_timestamp,"+
			"pokestop.updated, quest_type, quest_timestamp, quest_target, quest_conditions,"+
			"quest_rewards, quest_template, quest_title,"+
			"alternative_quest_type, alternative_quest_timestamp, alternative_quest_target,"+
			"alternative_quest_conditions, alternative_quest_rewards,"+
			"alternative_quest_template, alternative_quest_title, cell_id, lure_id, sponsor_id, partner_id,"+
			"ar_scan_eligible, power_up_points, power_up_level, power_up_end_timestamp "+
			"FROM pokestop "+
			"WHERE pokestop.id = ? ", fortId)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	pokestopCache.Set(fortId, pokestop, ttlcache.DefaultTTL)
	return &pokestop, nil
}

func hasChanges(old *Pokestop, new *Pokestop) bool {
	return !cmp.Equal(old, new, ignoreNearFloats)
}

var LureTime int64 = 1800

func (stop *Pokestop) updatePokestopFromFort(fortData *pogo.PokemonFortProto, cellId uint64) *Pokestop {
	stop.Id = fortData.FortId
	stop.Lat = fortData.Latitude
	stop.Lon = fortData.Longitude

	stop.PartnerId = null.NewString(fortData.PartnerId, fortData.PartnerId != "")
	stop.SponsorId = null.IntFrom(int64(fortData.Sponsor))
	stop.Enabled = null.BoolFrom(fortData.Enabled)
	stop.ArScanEligible = null.IntFrom(util.BoolToInt[int64](fortData.IsArScanEligible))
	stop.PowerUpPoints = null.IntFrom(int64(fortData.PowerUpProgressPoints))
	stop.PowerUpLevel, stop.PowerUpEndTimestamp = calculatePowerUpPoints(fortData)

	lastModifiedTimestamp := fortData.LastModifiedMs / 1000
	stop.LastModifiedTimestamp = null.IntFrom(lastModifiedTimestamp)

	if len(fortData.ActiveFortModifier) > 0 && int(fortData.ActiveFortModifier[0]) > 0 {
		lureId := int16(fortData.ActiveFortModifier[0])

		lureEnd := lastModifiedTimestamp + LureTime
		if stop.LureId != lureId {
			stop.LureExpireTimestamp = null.IntFrom(lureEnd)
			stop.LureId = lureId
		} else {
			now := time.Now().Unix()
			if now > lureEnd {
				// If a lure needs to be restarted
				stop.LureExpireTimestamp = null.IntFrom(lureEnd)
			}
		}
	}

	if fortData.ImageUrl != "" {
		stop.Url = null.StringFrom(fortData.ImageUrl)
	}
	stop.CellId = null.IntFrom(int64(cellId))

	return stop
}

func (stop *Pokestop) updatePokestopFromQuestProto(questProto *pogo.FortSearchOutProto, haveAr bool) {

	if questProto.ChallengeQuest == nil {
		log.Debugf("Received blank quest")
		return
	}
	questData := questProto.ChallengeQuest.Quest
	questTitle := questProto.ChallengeQuest.QuestDisplay.Title
	questType := int64(questData.QuestType)
	questTarget := int64(questData.Goal.Target)
	questTemplate := strings.ToLower(questData.TemplateId)

	conditions := []map[string]interface{}{}
	rewards := []map[string]interface{}{}

	for _, conditionData := range questData.Goal.Condition {
		condition := make(map[string]interface{})
		infoData := make(map[string]interface{})
		condition["type"] = int(conditionData.Type)
		switch conditionData.Type {
		case pogo.QuestConditionProto_WITH_BADGE_TYPE:
			info := conditionData.GetWithBadgeType()
			infoData["amount"] = info.Amount
			infoData["badge_rank"] = info.BadgeRank
			badgeTypeById := []int{}
			for _, badge := range info.BadgeType {
				badgeTypeById = append(badgeTypeById, int(badge))
			}
			infoData["badge_types"] = badgeTypeById

		case pogo.QuestConditionProto_WITH_ITEM:
			info := conditionData.GetWithItem()
			if int(info.Item) != 0 {
				infoData["item_id"] = int(info.Item)
			}
		case pogo.QuestConditionProto_WITH_RAID_LEVEL:
			info := conditionData.GetWithRaidLevel()
			raidLevelById := []int{}
			for _, raidLevel := range info.RaidLevel {
				raidLevelById = append(raidLevelById, int(raidLevel))
			}
			infoData["raid_levels"] = raidLevelById
		case pogo.QuestConditionProto_WITH_POKEMON_TYPE:
			info := conditionData.GetWithPokemonType()
			pokemonTypesById := []int{}
			for _, t := range info.PokemonType {
				pokemonTypesById = append(pokemonTypesById, int(t))
			}
			infoData["pokemon_type_ids"] = pokemonTypesById
		case pogo.QuestConditionProto_WITH_POKEMON_CATEGORY:
			info := conditionData.GetWithPokemonCategory()
			if info.CategoryName != "" {
				infoData["category_name"] = info.CategoryName
			}
			pokemonById := []int{}
			for _, pokemon := range info.PokemonIds {
				pokemonById = append(pokemonById, int(pokemon))
			}
			infoData["pokemon_ids"] = pokemonById
		case pogo.QuestConditionProto_WITH_WIN_RAID_STATUS:
		case pogo.QuestConditionProto_WITH_THROW_TYPE:
			info := conditionData.GetWithThrowType()
			if int(info.GetThrowType()) != 0 { // TODO: RDM has ThrowType here, ensure it is the same thing
				infoData["throw_type_id"] = int(info.GetThrowType())
			}
			infoData["hit"] = info.GetHit()
		case pogo.QuestConditionProto_WITH_THROW_TYPE_IN_A_ROW:
			info := conditionData.GetWithThrowType()
			if int(info.GetThrowType()) != 0 {
				infoData["throw_type_id"] = int(info.GetThrowType())
			}
			infoData["hit"] = info.GetHit()
		case pogo.QuestConditionProto_WITH_LOCATION:
			info := conditionData.GetWithLocation()
			infoData["cell_ids"] = info.S2CellId
		case pogo.QuestConditionProto_WITH_DISTANCE:
			info := conditionData.GetWithDistance()
			infoData["distance"] = info.DistanceKm
		case pogo.QuestConditionProto_WITH_POKEMON_ALIGNMENT:
			info := conditionData.GetWithPokemonAlignment()
			alignmentIds := []int{}
			for _, alignment := range info.Alignment {
				alignmentIds = append(alignmentIds, int(alignment))
			}
			infoData["alignment_ids"] = alignmentIds
		case pogo.QuestConditionProto_WITH_INVASION_CHARACTER:
			info := conditionData.GetWithInvasionCharacter()
			characterCategoryIds := []int{}
			for _, characterCategory := range info.Category {
				characterCategoryIds = append(characterCategoryIds, int(characterCategory))
			}
			infoData["character_category_ids"] = characterCategoryIds
		case pogo.QuestConditionProto_WITH_NPC_COMBAT:
			info := conditionData.GetWithNpcCombat()
			infoData["win"] = info.RequiresWin
			infoData["template_ids"] = info.CombatNpcTrainerId
		case pogo.QuestConditionProto_WITH_PLAYER_LEVEL:
			info := conditionData.GetWithPlayerLevel()
			infoData["level"] = info.Level
		case pogo.QuestConditionProto_WITH_BUDDY:
			info := conditionData.GetWithBuddy()
			infoData["min_buddy_level"] = int(info.MinBuddyLevel)
			infoData["must_be_on_map"] = info.MustBeOnMap
		case pogo.QuestConditionProto_WITH_DAILY_BUDDY_AFFECTION:
			info := conditionData.GetWithDailyBuddyAffection()
			infoData["min_buddy_affection_earned_today"] = info.MinBuddyAffectionEarnedToday
		case pogo.QuestConditionProto_WITH_TEMP_EVO_POKEMON:
			info := conditionData.GetWithTempEvoId()
			tempEvoIds := []int{}
			for _, evolution := range info.MegaForm {
				tempEvoIds = append(tempEvoIds, int(evolution))
			}
			infoData["raid_pokemon_evolutions"] = tempEvoIds
		case pogo.QuestConditionProto_WITH_ITEM_TYPE:
			info := conditionData.GetWithItemType()
			itemTypes := []int{}
			for _, itemType := range info.ItemType {
				itemTypes = append(itemTypes, int(itemType))
			}
			infoData["item_type_ids"] = itemTypes
		case pogo.QuestConditionProto_WITH_RAID_ELAPSED_TIME:
			info := conditionData.GetWithElapsedTime()
			infoData["time"] = int64(info.ElapsedTimeMs) / 1000
		case pogo.QuestConditionProto_WITH_WIN_GYM_BATTLE_STATUS:
		case pogo.QuestConditionProto_WITH_SUPER_EFFECTIVE_CHARGE:
		case pogo.QuestConditionProto_WITH_UNIQUE_POKESTOP:
		case pogo.QuestConditionProto_WITH_QUEST_CONTEXT:
		case pogo.QuestConditionProto_WITH_WIN_BATTLE_STATUS:
		case pogo.QuestConditionProto_WITH_CURVE_BALL:
		case pogo.QuestConditionProto_WITH_NEW_FRIEND:
		case pogo.QuestConditionProto_WITH_DAYS_IN_A_ROW:
		case pogo.QuestConditionProto_WITH_WEATHER_BOOST:
		case pogo.QuestConditionProto_WITH_DAILY_CAPTURE_BONUS:
		case pogo.QuestConditionProto_WITH_DAILY_SPIN_BONUS:
		case pogo.QuestConditionProto_WITH_UNIQUE_POKEMON:
		case pogo.QuestConditionProto_WITH_BUDDY_INTERESTING_POI:
		case pogo.QuestConditionProto_WITH_POKEMON_LEVEL:
		case pogo.QuestConditionProto_WITH_SINGLE_DAY:
		case pogo.QuestConditionProto_WITH_UNIQUE_POKEMON_TEAM:
		case pogo.QuestConditionProto_WITH_MAX_CP:
		case pogo.QuestConditionProto_WITH_LUCKY_POKEMON:
		case pogo.QuestConditionProto_WITH_LEGENDARY_POKEMON:
		case pogo.QuestConditionProto_WITH_GBL_RANK:
		case pogo.QuestConditionProto_WITH_CATCHES_IN_A_ROW:
		case pogo.QuestConditionProto_WITH_ENCOUNTER_TYPE:
		case pogo.QuestConditionProto_WITH_COMBAT_TYPE:
		case pogo.QuestConditionProto_WITH_GEOTARGETED_POI:
		case pogo.QuestConditionProto_WITH_FRIEND_LEVEL:
		case pogo.QuestConditionProto_WITH_STICKER:
		case pogo.QuestConditionProto_WITH_POKEMON_CP:
		case pogo.QuestConditionProto_WITH_RAID_LOCATION:
		case pogo.QuestConditionProto_WITH_FRIENDS_RAID:
		case pogo.QuestConditionProto_WITH_POKEMON_COSTUME:
		default:
			break
		}

		if infoData != nil {
			condition["info"] = infoData
		}
		conditions = append(conditions, condition)
	}

	for _, rewardData := range questData.QuestRewards {
		reward := make(map[string]interface{})
		infoData := make(map[string]interface{})
		reward["type"] = int(rewardData.Type)
		switch rewardData.Type {
		case pogo.QuestRewardProto_EXPERIENCE:
			infoData["amount"] = rewardData.GetExp()
		case pogo.QuestRewardProto_ITEM:
			info := rewardData.GetItem()
			infoData["amount"] = info.Amount
			infoData["item_id"] = int(info.Item)
		case pogo.QuestRewardProto_STARDUST:
			infoData["amount"] = rewardData.GetStardust()
		case pogo.QuestRewardProto_CANDY:
			info := rewardData.GetCandy()
			infoData["amount"] = info.Amount
			infoData["pokemon_id"] = int(info.PokemonId)
		case pogo.QuestRewardProto_XL_CANDY:
			info := rewardData.GetXlCandy()
			infoData["amount"] = info.Amount
			infoData["pokemon_id"] = int(info.PokemonId)
		case pogo.QuestRewardProto_POKEMON_ENCOUNTER:
			info := rewardData.GetPokemonEncounter()
			if info.IsHiddenDitto {
				infoData["pokemon_id"] = 132
				infoData["pokemon_id_display"] = int(info.GetPokemonId())
			} else {
				infoData["pokemon_id"] = int(info.GetPokemonId())
			}
			if info.PokemonDisplay != nil {
				infoData["costume_id"] = int(info.PokemonDisplay.Costume)
				infoData["form_id"] = int(info.PokemonDisplay.Form)
				infoData["gender_id"] = int(info.PokemonDisplay.Gender)
				infoData["shiny"] = info.PokemonDisplay.Shiny
			} else {

			}
		case pogo.QuestRewardProto_POKECOIN:
			infoData["amount"] = rewardData.GetPokecoin()
		case pogo.QuestRewardProto_STICKER:
			info := rewardData.GetSticker()
			infoData["amount"] = info.Amount
			infoData["sticker_id"] = info.StickerId
		case pogo.QuestRewardProto_MEGA_RESOURCE:
			info := rewardData.GetMegaResource()
			infoData["amount"] = info.Amount
			infoData["pokemon_id"] = int(info.PokemonId)
		case pogo.QuestRewardProto_AVATAR_CLOTHING:
		case pogo.QuestRewardProto_QUEST:
		case pogo.QuestRewardProto_LEVEL_CAP:
		case pogo.QuestRewardProto_INCIDENT:
		case pogo.QuestRewardProto_PLAYER_ATTRIBUTE:
		default:
			break

		}
		reward["info"] = infoData
		rewards = append(rewards, reward)
	}

	questConditions, _ := json.Marshal(conditions)
	questRewards, _ := json.Marshal(rewards)
	questTimestamp := time.Now().Unix()

	if !haveAr {
		stop.AlternativeQuestType = null.IntFrom(questType)
		stop.AlternativeQuestTarget = null.IntFrom(questTarget)
		stop.AlternativeQuestTemplate = null.StringFrom(questTemplate)
		stop.AlternativeQuestTitle = null.StringFrom(questTitle)
		stop.AlternativeQuestConditions = null.StringFrom(string(questConditions))
		stop.AlternativeQuestRewards = null.StringFrom(string(questRewards))
		stop.AlternativeQuestTimestamp = null.IntFrom(questTimestamp)
	} else {
		stop.QuestType = null.IntFrom(questType)
		stop.QuestTarget = null.IntFrom(questTarget)
		stop.QuestTemplate = null.StringFrom(questTemplate)
		stop.QuestTitle = null.StringFrom(questTitle)
		stop.QuestConditions = null.StringFrom(string(questConditions))
		stop.QuestRewards = null.StringFrom(string(questRewards))
		stop.QuestTimestamp = null.IntFrom(questTimestamp)
	}
}

func (stop *Pokestop) updatePokestopFromFortDetailsProto(fortData *pogo.FortDetailsOutProto) *Pokestop {
	stop.Id = fortData.Id
	stop.Lat = fortData.Latitude
	stop.Lon = fortData.Longitude
	if len(fortData.ImageUrl) > 0 {
		stop.Url = null.StringFrom(fortData.ImageUrl[0])
	}
	stop.Name = null.StringFrom(fortData.Name)

	return stop
}

func createPokestopWebhooks(oldStop *Pokestop, stop *Pokestop) {

	if stop.AlternativeQuestType.Valid && (oldStop == nil || stop.AlternativeQuestType != oldStop.AlternativeQuestType) {
		questHook := map[string]interface{}{
			"pokestop_id": stop.Id,
			"latitude":    stop.Lat,
			"longitude":   stop.Lon,
			"type":        stop.AlternativeQuestType,
			"target":      stop.AlternativeQuestTarget,
			"template":    stop.AlternativeQuestTemplate,
			"title":       stop.AlternativeQuestTarget,
			"conditions": func() []map[string]interface{} {
				var r []map[string]interface{}
				_ = json.Unmarshal([]byte(stop.AlternativeQuestConditions.ValueOrZero()), &r)
				return r
			}(),
			"rewards": func() []map[string]interface{} {
				var r []map[string]interface{}
				_ = json.Unmarshal([]byte(stop.AlternativeQuestRewards.ValueOrZero()), &r)
				return r
			}(),
			"updated": stop.Updated,
			"pokestop_name": func() string {
				if stop.Name.Valid {
					return stop.Name.String
				} else {
					return "Unknown"
				}
			}(),
			"ar_scan_eligible": stop.ArScanEligible.ValueOrZero(),
			"pokestop_url":     stop.Url.Valid,
			"with_ar":          false,
		}
		webhooks.AddMessage(webhooks.Quest, questHook)
	}

	if stop.QuestType.Valid && (oldStop == nil || stop.QuestType != oldStop.QuestType) {
		questHook := map[string]interface{}{
			"pokestop_id": stop.Id,
			"latitude":    stop.Lat,
			"longitude":   stop.Lon,
			"type":        stop.QuestType,
			"target":      stop.QuestTarget,
			"template":    stop.QuestTemplate,
			"title":       stop.QuestTarget,
			"conditions": func() []map[string]interface{} {
				var r []map[string]interface{}
				_ = json.Unmarshal([]byte(stop.QuestConditions.ValueOrZero()), &r)
				return r
			}(),
			"rewards": func() []map[string]interface{} {
				var r []map[string]interface{}
				_ = json.Unmarshal([]byte(stop.QuestRewards.ValueOrZero()), &r)
				return r
			}(),
			"updated": stop.Updated,
			"pokestop_name": func() string {
				if stop.Name.Valid {
					return stop.Name.String
				} else {
					return "Unknown"
				}
			}(),
			"ar_scan_eligible": stop.ArScanEligible.ValueOrZero(),
			"pokestop_url":     stop.Url.Valid,
			"with_ar":          true,
		}
		webhooks.AddMessage(webhooks.Quest, questHook)
	}
	if (oldStop == nil && (stop.LureId != 0 || stop.PowerUpEndTimestamp.ValueOrZero() != 0)) || (oldStop != nil && ((stop.LureExpireTimestamp != oldStop.LureExpireTimestamp && stop.LureId != 0) || stop.PowerUpEndTimestamp != oldStop.PowerUpEndTimestamp)) {
		pokestopHook := map[string]interface{}{
			"pokestop_id": stop.Id,
			"latitude":    stop.Lat,
			"longitude":   stop.Lon,
			"name": func() string {
				if stop.Name.Valid {
					return stop.Name.String
				} else {
					return "Unknown"
				}
			}(),
			"url":                    stop.Url.ValueOrZero(),
			"lure_expiration":        stop.LureExpireTimestamp.ValueOrZero(),
			"last_modified":          stop.LastModifiedTimestamp.ValueOrZero(),
			"enabled":                stop.Enabled.ValueOrZero(),
			"lure_id":                stop.LureId,
			"ar_scan_eligible":       stop.ArScanEligible.ValueOrZero(),
			"power_up_level":         stop.PowerUpLevel.ValueOrZero(),
			"power_up_points":        stop.PowerUpPoints.ValueOrZero(),
			"power_up_end_timestamp": stop.PowerUpPoints.ValueOrZero(),
			"updated":                stop.Updated,
		}

		webhooks.AddMessage(webhooks.Pokestop, pokestopHook)
	}
}

func savePokestopRecord(db *sqlx.DB, pokestop *Pokestop) {
	oldPokestop, _ := getPokestopRecord(db, pokestop.Id)

	if oldPokestop != nil && !hasChanges(oldPokestop, pokestop) {
		return
	}

	log.Traceln(cmp.Diff(oldPokestop, pokestop))

	if oldPokestop == nil {
		res, err := db.NamedExec(
			"INSERT INTO pokestop ("+
				"id, lat, lon, name, url, enabled, lure_expire_timestamp, last_modified_timestamp, quest_type,"+
				"quest_timestamp, quest_target, quest_conditions, quest_rewards, quest_template, quest_title,"+
				"alternative_quest_type, alternative_quest_timestamp, alternative_quest_target,"+
				"alternative_quest_conditions, alternative_quest_rewards, alternative_quest_template,"+
				"alternative_quest_title, cell_id, lure_id, sponsor_id, partner_id, ar_scan_eligible,"+
				"power_up_points, power_up_level, power_up_end_timestamp, updated, first_seen_timestamp)"+
				"VALUES ("+
				":id, :lat, :lon, :name, :url, :enabled, :lure_expire_timestamp, :last_modified_timestamp, :quest_type,"+
				":quest_timestamp, :quest_target, :quest_conditions, :quest_rewards, :quest_template, :quest_title,"+
				":alternative_quest_type, :alternative_quest_timestamp, :alternative_quest_target,"+
				":alternative_quest_conditions, :alternative_quest_rewards, :alternative_quest_template,"+
				":alternative_quest_title, :cell_id, :lure_id, :sponsor_id, :partner_id, :ar_scan_eligible,"+
				":power_up_points, :power_up_level, :power_up_end_timestamp,"+
				"UNIX_TIMESTAMP(), UNIX_TIMESTAMP() )",
			pokestop)

		if err != nil {
			log.Errorf("insert pokestop: %s", err)
			return
		}
		_ = res
	} else {
		res, err := db.NamedExec(
			"UPDATE pokestop SET "+
				"lat = :lat,"+
				"lon = :lon,"+
				"name = :name,"+
				"url = :url,"+
				"enabled = :enabled,"+
				"lure_expire_timestamp = :lure_expire_timestamp,"+
				"last_modified_timestamp = :last_modified_timestamp,"+
				"updated = UNIX_TIMESTAMP(),"+
				"quest_type = :quest_type, "+
				"quest_timestamp = :quest_timestamp, "+
				"quest_target = :quest_target, "+
				"quest_conditions = :quest_conditions, "+
				"quest_rewards = :quest_rewards, "+
				"quest_template = :quest_template, "+
				"quest_title = :quest_title,"+
				"alternative_quest_type = :alternative_quest_type, "+
				"alternative_quest_timestamp = :alternative_quest_timestamp,"+
				"alternative_quest_target = :alternative_quest_target, "+
				"alternative_quest_conditions = :alternative_quest_conditions, "+
				"alternative_quest_rewards = :alternative_quest_rewards,"+
				" alternative_quest_template = :alternative_quest_template,"+
				"alternative_quest_title = :alternative_quest_title,"+
				"cell_id = :cell_id,"+
				"lure_id = :lure_id,"+
				"deleted = false,"+
				"sponsor_id = :sponsor_id,"+
				"partner_id = :partner_id,"+
				"ar_scan_eligible = :ar_scan_eligible,"+
				"power_up_points = :power_up_points,"+
				"power_up_level = :power_up_level,"+
				"power_up_end_timestamp = :power_up_end_timestamp"+
				" WHERE id = :id",
			pokestop,
		)
		if err != nil {
			log.Errorf("update pokestop: %s", err)
			return
		}
		_ = res
	}
	pokestopCache.Set(pokestop.Id, *pokestop, ttlcache.DefaultTTL)
	createPokestopWebhooks(oldPokestop, pokestop)
}

func UpdatePokestopRecordWithFortDetailsOutProto(db *sqlx.DB, fort *pogo.FortDetailsOutProto) string {
	pokestop, err := getPokestopRecord(db, fort.Id) // should check error
	if err != nil {
		log.Printf("Update pokestop %s", err)
		return fmt.Sprintf("Error %s", err)
	}

	if pokestop == nil {
		pokestop = &Pokestop{}
	}
	pokestop.updatePokestopFromFortDetailsProto(fort)
	savePokestopRecord(db, pokestop)
	return fmt.Sprintf("%s %s", fort.Id, fort.Name)
}

func UpdatePokestopWithQuest(db *sqlx.DB, quest *pogo.FortSearchOutProto, haveAr bool) string {
	if quest.ChallengeQuest == nil {
		return "No quest"
	}

	pokestop, err := getPokestopRecord(db, quest.FortId)
	if err != nil {
		log.Printf("Update quest %s", err)
		return fmt.Sprintf("error %s", err)
	}

	if pokestop == nil {
		pokestop = &Pokestop{}
	}
	pokestop.updatePokestopFromQuestProto(quest, haveAr)
	savePokestopRecord(db, pokestop)
	return fmt.Sprintf("%s", quest.FortId)
}