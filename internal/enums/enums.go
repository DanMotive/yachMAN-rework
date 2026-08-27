package enums

// ── Skill Directions (20) ─────────────────────────────────────────────

type SkillDirection string

const (
	DirMining           SkillDirection = "добыча"
	DirForest           SkillDirection = "лес"
	DirFuel             SkillDirection = "топливо"
	DirEnergy           SkillDirection = "энергетика"
	DirMetallurgy       SkillDirection = "металлургия"
	DirConstruction     SkillDirection = "строительство"
	DirChemistry        SkillDirection = "химия"
	DirIT               SkillDirection = "IT"
	DirTrade            SkillDirection = "торговля"
	DirAgro             SkillDirection = "агро"
	DirTransport        SkillDirection = "транспорт"
	DirFood             SkillDirection = "питание"
	DirRepair           SkillDirection = "ремонт"
	DirMedicine         SkillDirection = "медицина"
	DirEducation        SkillDirection = "образование"
	DirScience          SkillDirection = "наука"
	DirSecurity         SkillDirection = "безопасность"
	DirMedia            SkillDirection = "медиа"
	DirCommunal         SkillDirection = "коммунальные услуги"
	DirRecycling        SkillDirection = "переработка"
)

var AllSkillDirections = []SkillDirection{
	DirMining, DirForest, DirFuel, DirEnergy, DirMetallurgy,
	DirConstruction, DirChemistry, DirIT, DirTrade, DirAgro,
	DirTransport, DirFood, DirRepair, DirMedicine, DirEducation,
	DirScience, DirSecurity, DirMedia, DirCommunal, DirRecycling,
}

// ── Resources (10) ────────────────────────────────────────────────────

type ResourceType string

const (
	R1Food           ResourceType = "R1" // Продовольствие
	R2Ore            ResourceType = "R2" // Руда
	R3Wood           ResourceType = "R3" // Древесина
	R4Fuel           ResourceType = "R4" // Топливо
	R5Energy         ResourceType = "R5" // Энергия
	R6Metal          ResourceType = "R6" // Металл
	R7Materials      ResourceType = "R7" // Материалы
	R8Chemicals      ResourceType = "R8" // Химикаты
	R9Tech           ResourceType = "R9" // Технологии
	R10ConsumerGoods ResourceType = "R10" // Потребительские товары
)

var BaseResourcePrices = map[ResourceType]int{
	R1Food: 18, R2Ore: 30, R3Wood: 22, R4Fuel: 40, R5Energy: 16,
	R6Metal: 55, R7Materials: 35, R8Chemicals: 48, R9Tech: 75, R10ConsumerGoods: 95,
}

// ── City Levels ───────────────────────────────────────────────────────

type CityLevel string

const (
	LevelCommunity    CityLevel = "community"     // Община
	LevelVillage      CityLevel = "village"       // Деревня
	LevelSettlement   CityLevel = "settlement"    // Посёлок
	LevelRural        CityLevel = "rural"         // Село
	LevelUrbanType    CityLevel = "urban_type"    // ПГТ
	LevelSmallCity    CityLevel = "small_city"    // Малый город
	LevelCity         CityLevel = "city"          // Город
	LevelBigCity      CityLevel = "big_city"      // Большой город
	LevelMillionCity  CityLevel = "million_city"  // Город-миллионник
	LevelMetropolis   CityLevel = "metropolis"    // Метрополис
	LevelMegalopolis  CityLevel = "megalopolis"   // Мегаполис
)

type CityLevelSpec struct {
	Level            CityLevel
	NPCPopulation    int
	DP               int
	AdminSlots       int
	MajorProjectSlots int
	ContractLimit    int
}

var CityLevels = []CityLevelSpec{
	{LevelCommunity, 0, 0, 3, 1, 1},
	{LevelVillage, 2500, 5, 5, 1, 1},
	{LevelSettlement, 10000, 15, 8, 2, 1},
	{LevelRural, 25000, 30, 12, 2, 2},
	{LevelUrbanType, 50000, 50, 18, 3, 2},
	{LevelSmallCity, 100000, 80, 26, 4, 3},
	{LevelCity, 200000, 120, 36, 5, 4},
	{LevelBigCity, 400000, 180, 50, 7, 5},
	{LevelMillionCity, 1000000, 260, 70, 9, 7},
	{LevelMetropolis, 2500000, 360, 95, 12, 10},
	{LevelMegalopolis, 5000000, 500, 130, 16, 14},
}

// ── Access Modes ──────────────────────────────────────────────────────

type AccessMode string

const (
	AccessOpen      AccessMode = "open"
	AccessModerated AccessMode = "moderated"
	AccessClosed    AccessMode = "closed"
)

// ── Corporation Roles ─────────────────────────────────────────────────

type CorpRole string

const (
	CorpOwner     CorpRole = "owner"
	CorpExecutive CorpRole = "executive"
	CorpManager   CorpRole = "manager"
	CorpEmployee  CorpRole = "employee"
)

// ── Event Types ───────────────────────────────────────────────────────

type EventType string

const (
	EventWorld    EventType = "world"
	EventCity     EventType = "city"
	EventEconomic EventType = "economic"
	EventSocial   EventType = "social"
)

// ── Order Types / Status ──────────────────────────────────────────────

type OrderType string

const (
	OrderBuy  OrderType = "buy"
	OrderSell OrderType = "sell"
)

type OrderStatus string

const (
	OrderOpen    OrderStatus = "open"
	OrderFilled  OrderStatus = "filled"
	OrderCancel  OrderStatus = "cancelled"
)

// ── Ledger Entity Types ───────────────────────────────────────────────

type LedgerEntity string

const (
	LedgerUser         LedgerEntity = "user"
	LedgerCity         LedgerEntity = "city"
	LedgerCorporation  LedgerEntity = "corporation"
)

// ── Education Extra Directions (non-work skill directions used in edu) ─

type EduDirection string

const (
	EduGeneral        EduDirection = "Общее"
	EduProduction     EduDirection = "Производство"
	EduEngineering    EduDirection = "Инженерия"
	EduEconomics      EduDirection = "Экономика"
	EduSpace          EduDirection = "Космос"
)

// ── VIP Plans ─────────────────────────────────────────────────────────

type VIPPlan string

const (
	VIP30d  VIPPlan = "30d"
	VIP90d  VIPPlan = "90d"
	VIP365d VIPPlan = "365d"
)

var VIPPrices = map[VIPPlan]int{
	VIP30d:  100,
	VIP90d:  250,
	VIP365d: 800,
}

// ── Notification Types ────────────────────────────────────────────────

type NotificationType string

const (
	NotifWorkComplete  NotificationType = "work_complete"
	NotifStudyReady    NotificationType = "study_ready"
	NotifDailyBonus    NotificationType = "daily_bonus"
	NotifSalary        NotificationType = "salary"
	NotifEvent         NotificationType = "event"
	NotifTradeContract NotificationType = "trade_contract"
	NotifCorporate     NotificationType = "corporate"
)

// ── City Admin Positions ──────────────────────────────────────────────

const (
	AdminMayor       = "mayor"
	AdminDeputy      = "deputy"
	AdminTreasurer   = "treasurer"
	AdminEconomist   = "economist"
	AdminModerator   = "moderator"
)
