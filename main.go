package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const dataFile = "data/players.json"

type Pokemon struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Level    int    `json:"level"`
	XP       int    `json:"xp"`
	XPToNext int    `json:"xp_to_next"`
	Shiny    bool   `json:"shiny"`
	HP       int    `json:"hp"`
}

type Move struct {
	Name   string
	Type   string
	Power  int
}

type Player struct {
	DiscordID    string    `json:"discord_id"`
	Username     string    `json:"username"`
	Coins        int       `json:"coins"`
	Level        int       `json:"level"`
	XP           int       `json:"xp"`
	Pokemon         []Pokemon       `json:"pokemon"`
	CurrentRoute    string         `json:"current_route"`
	Items           map[string]int `json:"items"`
	ActiveLureUntil time.Time      `json:"active_lure_until"`
	ActiveLureName  string         `json:"active_lure_name"`
	ColiseumWins    int            `json:"coliseum_wins"`
	Team            []int          `json:"team"`
	Badges          []string       `json:"badges"`
	LeagueWins      int            `json:"league_wins"`
	Champion        bool           `json:"champion"`
}

type Route struct {
	ID          string
	Name        string
	Description string
	UnlockLevel int
	MinLevel    int
	MaxLevel    int
	PokemonIDs  []int
}

type Evolution struct {
	FromID int
	FromName string
	ToID int
	ToName string
	Level int
}


type Encounter struct {
	OwnerID   string
	Pokemon   Pokemon
	ExpiresAt time.Time
}

type Battle struct {
	OwnerID       string
	PlayerPokemon Pokemon
	Opponent      Pokemon
	PlayerHP      int
	OpponentHP    int
	GymID         string
	Elite         bool
	EliteIndex    int
}

type Store struct {
	mu         sync.RWMutex
	Players    map[string]Player
	Encounters map[string]Encounter
	Battles    map[string]Battle
}

var store = &Store{Players: map[string]Player{}, Encounters: map[string]Encounter{}, Battles: map[string]Battle{}}

var pokemonPool = []Pokemon{
	{ID: 1, Name: "Bulbasaur", Level: 1, XPToNext: 14},
	{ID: 4, Name: "Charmander", Level: 1, XPToNext: 14},
	{ID: 7, Name: "Squirtle", Level: 1, XPToNext: 14},
	{ID: 10, Name: "Caterpie", Level: 1, XPToNext: 14},
	{ID: 13, Name: "Weedle", Level: 1, XPToNext: 14},
	{ID: 16, Name: "Pidgey", Level: 1, XPToNext: 14},
	{ID: 19, Name: "Rattata", Level: 1, XPToNext: 14},
	{ID: 21, Name: "Spearow", Level: 1, XPToNext: 14},
	{ID: 25, Name: "Pikachu", Level: 1, XPToNext: 14},
	{ID: 43, Name: "Oddish", Level: 1, XPToNext: 14},
	{ID: 52, Name: "Meowth", Level: 1, XPToNext: 14},
	{ID: 41, Name: "Zubat", Level: 1, XPToNext: 14},
	{ID: 74, Name: "Geodude", Level: 1, XPToNext: 14},
	{ID: 35, Name: "Clefairy", Level: 1, XPToNext: 14},
	{ID: 29, Name: "Nidoran♀", Level: 1, XPToNext: 14},
	{ID: 32, Name: "Nidoran♂", Level: 1, XPToNext: 14},
	{ID: 102, Name: "Exeggcute", Level: 1, XPToNext: 14},
	{ID: 128, Name: "Tauros", Level: 1, XPToNext: 14},
	{ID: 123, Name: "Scyther", Level: 1, XPToNext: 14},
	{ID: 144, Name: "Articuno", Level: 30, XPToNext: 14},
	{ID: 145, Name: "Zapdos", Level: 30, XPToNext: 14},
	{ID: 146, Name: "Moltres", Level: 30, XPToNext: 14},
	{ID: 150, Name: "Mewtwo", Level: 50, XPToNext: 14},
	{ID: 151, Name: "Mew", Level: 40, XPToNext: 14},
}


type Gym struct {
	ID string
	Name string
	Leader string
	Type string
	Badge string
	UnlockLevel int
	RewardCoins int
	Pokemon []Pokemon
}

var gyms = []Gym{
	{ID:"pedra", Name:"Ginásio de Pewter", Leader:"Brock", Type:"Pedra", Badge:"🌑 Insígnia Boulder", UnlockLevel:3, RewardCoins:100, Pokemon:[]Pokemon{{ID:74,Name:"Geodude",Level:5,XPToNext:30},{ID:95,Name:"Onix",Level:7,XPToNext:30}}},
	{ID:"agua", Name:"Ginásio de Cerulean", Leader:"Misty", Type:"Água", Badge:"💧 Insígnia Cascade", UnlockLevel:6, RewardCoins:150, Pokemon:[]Pokemon{{ID:120,Name:"Staryu",Level:8,XPToNext:30},{ID:121,Name:"Starmie",Level:10,XPToNext:30}}},
	{ID:"eletrico", Name:"Ginásio de Vermilion", Leader:"Lt. Surge", Type:"Elétrico", Badge:"⚡ Insígnia Thunder", UnlockLevel:9, RewardCoins:200, Pokemon:[]Pokemon{{ID:100,Name:"Voltorb",Level:11,XPToNext:30},{ID:26,Name:"Raichu",Level:13,XPToNext:30}}},
	{ID:"planta", Name:"Ginásio de Celadon", Leader:"Erika", Type:"Planta", Badge:"🌈 Insígnia Rainbow", UnlockLevel:12, RewardCoins:250, Pokemon:[]Pokemon{{ID:70,Name:"Weepinbell",Level:14,XPToNext:30},{ID:45,Name:"Vileplume",Level:16,XPToNext:30}}},
	{ID:"veneno", Name:"Ginásio de Fuchsia", Leader:"Koga", Type:"Veneno", Badge:"🧪 Insígnia Soul", UnlockLevel:15, RewardCoins:300, Pokemon:[]Pokemon{{ID:109,Name:"Koffing",Level:17,XPToNext:30},{ID:110,Name:"Weezing",Level:19,XPToNext:30}}},
	{ID:"psiquico", Name:"Ginásio de Saffron", Leader:"Sabrina", Type:"Psíquico", Badge:"🧠 Insígnia Marsh", UnlockLevel:18, RewardCoins:350, Pokemon:[]Pokemon{{ID:64,Name:"Kadabra",Level:20,XPToNext:30},{ID:65,Name:"Alakazam",Level:22,XPToNext:30}}},
	{ID:"fogo", Name:"Ginásio de Cinnabar", Leader:"Blaine", Type:"Fogo", Badge:"🔥 Insígnia Volcano", UnlockLevel:21, RewardCoins:400, Pokemon:[]Pokemon{{ID:58,Name:"Growlithe",Level:23,XPToNext:30},{ID:78,Name:"Rapidash",Level:25,XPToNext:30}}},
	{ID:"terra", Name:"Ginásio de Viridian", Leader:"Giovanni", Type:"Terra", Badge:"🌍 Insígnia Earth", UnlockLevel:24, RewardCoins:500, Pokemon:[]Pokemon{{ID:111,Name:"Rhyhorn",Level:26,XPToNext:30},{ID:112,Name:"Rhydon",Level:28,XPToNext:30}}},
}

type LeagueTrainer struct {
	Name string
	Type string
	Pokemon []Pokemon
	IsChampion bool
}

var leagueTrainers = []LeagueTrainer{
	{Name:"Lorelei", Type:"Gelo", Pokemon:[]Pokemon{{ID:87,Name:"Dewgong",Level:54,XPToNext:30},{ID:91,Name:"Cloyster",Level:55,XPToNext:30},{ID:124,Name:"Jynx",Level:56,XPToNext:30}}},
	{Name:"Bruno", Type:"Lutador", Pokemon:[]Pokemon{{ID:95,Name:"Onix",Level:55,XPToNext:30},{ID:106,Name:"Hitmonlee",Level:56,XPToNext:30},{ID:68,Name:"Machamp",Level:58,XPToNext:30}}},
	{Name:"Agatha", Type:"Fantasma", Pokemon:[]Pokemon{{ID:94,Name:"Gengar",Level:56,XPToNext:30},{ID:93,Name:"Haunter",Level:57,XPToNext:30},{ID:94,Name:"Gengar",Level:60,XPToNext:30}}},
	{Name:"Lance", Type:"Dragão", Pokemon:[]Pokemon{{ID:130,Name:"Gyarados",Level:58,XPToNext:30},{ID:142,Name:"Aerodactyl",Level:60,XPToNext:30},{ID:149,Name:"Dragonite",Level:62,XPToNext:30}}},
	{Name:"Campeão Blue", Type:"Misto", IsChampion:true, Pokemon:[]Pokemon{{ID:18,Name:"Pidgeot",Level:61,XPToNext:30},{ID:65,Name:"Alakazam",Level:59,XPToNext:30},{ID:112,Name:"Rhydon",Level:61,XPToNext:30},{ID:130,Name:"Gyarados",Level:63,XPToNext:30},{ID:6,Name:"Charizard",Level:65,XPToNext:30}}},
}

var routes = []Route{
	{ID: "route1", Name: "Route 1", Description: "Uma rota tranquila para começar sua jornada.", UnlockLevel: 1, MinLevel: 1, MaxLevel: 5, PokemonIDs: []int{16, 19, 21, 43, 25}},
	{ID: "route2", Name: "Route 2", Description: "Uma rota com Pokémon um pouco mais fortes.", UnlockLevel: 3, MinLevel: 3, MaxLevel: 7, PokemonIDs: []int{16, 19, 21, 25, 52}},
	{ID: "viridian", Name: "Viridian Forest", Description: "Uma floresta cheia de Pokémon do tipo Inseto.", UnlockLevel: 5, MinLevel: 5, MaxLevel: 9, PokemonIDs: []int{10, 13, 25, 1}},
	{ID: "moon", Name: "Mt. Moon", Description: "Uma caverna misteriosa com Pokémon raros.", UnlockLevel: 8, MinLevel: 8, MaxLevel: 12, PokemonIDs: []int{41, 74, 35, 52}},
	{ID: "safari", Name: "Safari Zone", Description: "Uma área especial com encontros muito raros.", UnlockLevel: 12, MinLevel: 12, MaxLevel: 18, PokemonIDs: []int{29, 32, 102, 128, 123}},
}


func main() {
	// Load local .env when present. Environment variables still take precedence.
	_ = godotenv.Load()

	if err := loadStore(); err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN is not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	dg.AddHandler(onReady)
	dg.AddHandler(onInteraction)
	dg.Identify.Intents = discordgo.IntentsGuilds

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}
	if err := registerCommands(dg); err != nil {
		log.Fatal(err)
	}

	log.Println("PokeDiscord is online.")
	select {}
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s#%s", r.User.Username, r.User.Discriminator)
}

func registerCommands(s *discordgo.Session) error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name: "iniciar", Description: "Comece sua jornada Pokémon",
			Options: []*discordgo.ApplicationCommandOption{{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "starter", Description: "Escolha seu Pokémon inicial", Required: false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Bulbasaur", Value: "Bulbasaur"},
					{Name: "Charmander", Value: "Charmander"},
					{Name: "Squirtle", Value: "Squirtle"},
				},
			}},
		},
		{Name: "perfil", Description: "Veja seu perfil de treinador"},
		{Name: "pokemon", Description: "Veja seus Pokémon"},
		{Name: "treinar", Description: "Treine um Pokémon da sua coleção", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "numero", Description: "Número do Pokémon em /pokemon", Required: true, MinValue: func() *float64 { v := 1.0; return &v }()}}},
		{Name: "pokedex", Description: "Veja sua Pokédex e o progresso das rotas"},
		{Name: "inventario", Description: "Veja seus itens e iscas"},
		{Name: "loja", Description: "Veja os itens disponíveis na loja"},
		{Name: "comprar", Description: "Compre um item na loja", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "item", Description: "Item que deseja comprar", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Isca", Value: "isca"}, {Name: "Super Isca", Value: "super_isca"}}}}},
		{Name: "usar", Description: "Use uma isca do inventário", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "item", Description: "Isca que deseja usar", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Isca", Value: "isca"}, {Name: "Super Isca", Value: "super_isca"}}}}},
		{Name: "coliseu", Description: "Entre no Coliseu e enfrente um adversário", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "numero", Description: "Número do Pokémon em /pokemon", Required: true, MinValue: func() *float64 { v := 1.0; return &v }()}}},
		{Name: "liga", Description: "Veja sua progressão no Coliseu"},
		{Name: "ranking", Description: "Veja o ranking de treinadores"},
		{Name: "curar", Description: "Recupere o HP de todos os seus Pokémon"},
		{Name: "equipe", Description: "Monte sua equipe de até 6 Pokémon", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "ver", Description: "Veja sua equipe"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "adicionar", Description: "Adicione um Pokémon à equipe", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "numero", Description: "Número do Pokémon em /pokemon", Required: true}}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remover", Description: "Remova um Pokémon da equipe", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "numero", Description: "Número do Pokémon em /pokemon", Required: true}}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "limpar", Description: "Remova todos os Pokémon da equipe"},
		}},
		{Name: "ginasios", Description: "Veja os ginásios e suas insígnias"},
		{Name: "elite", Description: "Veja a Elite Four e o Campeão"},
		{Name: "ginasio", Description: "Desafie um líder de ginásio", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "numero", Description: "Número do ginásio", Required: true}}},
		{Name: "procurar", Description: "Procure um Pokémon selvagem"},
		{Name: "lendarios", Description: "Veja os Pokémon lendários e como encontrá-los"},
		{Name: "rotas", Description: "Veja as rotas e escolha onde caçar Pokémon"},
		{Name: "capturar", Description: "Tente capturar o Pokémon encontrado"},
		{Name: "fugir", Description: "Fuja do encontro atual"},
	}
	guildID := strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID"))
	if guildID == "" {
		return fmt.Errorf("DISCORD_GUILD_ID is not set")
	}

	log.Printf("Discord bot ID: %s", s.State.User.ID)
	log.Printf("Configured guild ID: %s", guildID)

	guild, err := s.Guild(guildID)
	if err != nil {
		return fmt.Errorf("cannot access configured guild %s: %w; check that this is the correct Server ID and that PokeDisc is installed there", guildID, err)
	}
	log.Printf("Guild access OK: %s (%s)", guild.Name, guild.ID)

	registered, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, commands)
	if err != nil {
		return fmt.Errorf("register slash commands in guild %s: %w", guildID, err)
	}

	log.Printf("Registered %d slash commands in %s", len(registered), guild.Name)
	return nil
}

func onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleCommand(s, i)
	case discordgo.InteractionMessageComponent:
		handleComponent(s, i)
	}
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "iniciar":
		handleStart(s, i)
	case "perfil":
		handleProfile(s, i)
	case "pokemon":
		handlePokemon(s, i)
	case "treinar":
		handleTrain(s, i)
	case "pokedex":
		handlePokedex(s, i)
	case "inventario":
		handleInventory(s, i)
	case "loja":
		handleShop(s, i)
	case "comprar":
		handleBuy(s, i)
	case "usar":
		handleUse(s, i)
	case "coliseu":
		handleColiseum(s, i)
	case "liga":
		handleLeague(s, i)
	case "ranking":
		handleRanking(s, i)
	case "curar":
		handleHeal(s, i)
	case "equipe":
		handleTeam(s, i)
	case "ginasios":
		handleGyms(s, i)
	case "ginasio":
		handleGymBattle(s, i)
	case "elite":
		handleElite(s, i)
	case "procurar":
		handleHunt(s, i)
	case "lendarios":
		handleLegendaries(s, i)
	case "rotas":
		handleRoutes(s, i)
	case "capturar":
		handleCatch(s, i)
	case "fugir":
		handleFlee(s, i)
	}
}

func handleStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := iUser(i)
	starter := "Charmander"
	if options := i.ApplicationCommandData().Options; len(options) > 0 && options[0].StringValue() != "" {
		starter = options[0].StringValue()
	}

	store.mu.Lock()
	if _, exists := store.Players[user.ID]; exists {
		store.mu.Unlock()
		respond(s, i, "Você já começou sua jornada! Use `/perfil`.")
		return
	}
	p := Player{
		DiscordID: user.ID, Username: user.Username, Coins: 100, Level: 1,
		Pokemon: []Pokemon{newPokemonByName(starter)},
		CurrentRoute: "route1",
		Items: map[string]int{"isca": 0, "super_isca": 0},
		Team: []int{0},
		Badges: []string{},
		LeagueWins: 0,
		Champion: false,
	}
	store.Players[user.ID] = p
	store.mu.Unlock()
	_ = saveStore()

	pk := newPokemonByName(starter)
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "🎒 Sua jornada começou!",
		Description: fmt.Sprintf("Bem-vinda, **%s**! Seu parceiro inicial é **%s**.", user.Username, starter),
		Color: 0xF2A900,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: spriteURL(pk.ID)},
		Image: &discordgo.MessageEmbedImage{URL: officialArtworkURL(pk.ID)},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Próximo passo", Value: "Use `/procurar` para procurar um Pokémon selvagem."},
			{Name: "💰 Coins", Value: "100"},
		},
	})
}


func handleProfile(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p,ok:=getPlayer(iUser(i).ID);if !ok{respond(s,i,"Você ainda não começou. Use /iniciar primeiro.");return}
	first:=Pokemon{ID:25,Name:"Pikachu",Level:1};if len(p.Pokemon)>0{first=p.Pokemon[0]};first.Type=pokemonType(first.ID)
	title:="👤 Seu Perfil";if p.Champion{title="👑 Seu Perfil • CAMPEÃO"}
	desc:=fmt.Sprintf("**%s**\nTreinador Pokémon",p.Username);if p.Champion{desc+=" • 👑 **Campeão da Liga**"}
	xpNext:=p.Level*50;if xpNext<1{xpNext=1}
	respondEmbed(s,i,&discordgo.MessageEmbed{Title:title,Description:desc,Color:0x5865F2,Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(first.ID)},Fields:[]*discordgo.MessageEmbedField{
		{Name:"⭐ Level",Value:fmt.Sprintf("**%d**\n%s\n%d/%d XP",p.Level,progressBar(p.XP,xpNext,10),p.XP,xpNext),Inline:true},
		{Name:"💰 Coins",Value:fmt.Sprintf("**%d**",p.Coins),Inline:true},
		{Name:"📦 Pokémon",Value:fmt.Sprintf("**%d** capturados",len(p.Pokemon)),Inline:true},
		{Name:"🏆 Coliseu",Value:fmt.Sprintf("**%d** vitórias",p.ColiseumWins),Inline:true},
		{Name:"🏅 Insígnias",Value:fmt.Sprintf("**%d/8**\n%s",len(p.Badges),strings.Repeat("🏅",len(p.Badges))),Inline:true},
		{Name:"🏆 Liga",Value:fmt.Sprintf("%s\n**%d/5** desafios",progressBar(p.LeagueWins,5,10),p.LeagueWins),Inline:true},
		{Name:"👑 Título",Value:func()string{if p.Champion{return "**CAMPEÃO**"};return "Treinador"}(),Inline:true},
		{Name:"🔥 Parceiro",Value:fmt.Sprintf("**%s**\n%s · Lv. %d",first.Name,typeEmoji(first.Type),first.Level),Inline:true},
	},Footer:&discordgo.MessageEmbedFooter{Text:"/pokemon • /equipe • /ginasios • /elite"}})
}


func handlePokemon(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p,ok:=getPlayer(iUser(i).ID);if !ok{respond(s,i,"Você ainda não começou. Use /iniciar primeiro.");return}
	if len(p.Pokemon)==0{respond(s,i,"📦 Sua coleção está vazia.");return}
	fields:=make([]*discordgo.MessageEmbedField,0,len(p.Pokemon))
	for idx,pk:=range p.Pokemon{
		pk.Type=pokemonType(pk.ID);hp:=pk.HP;if hp<=0||hp>battleHP(pk){hp=battleHP(pk)}
		teamPos:="-";for pos,teamIdx:=range p.Team{if teamIdx==idx{teamPos=strconv.Itoa(pos+1);break}}
		evoText:="";if evo:=evolutionFor(pk);evo!=nil{evoText=fmt.Sprintf("\n↗️ Evolui no Lv. %d → %s",evo.Level,evo.ToName)}
		teamText:="▫️ Fora da equipe";if teamPos!="-" {teamText="👥 Equipe #"+teamPos}
		shiny:="";if pk.Shiny{shiny=" ✨"}
		fields=append(fields,&discordgo.MessageEmbedField{Name:fmt.Sprintf("**%d. %s**%s",idx+1,pk.Name,shiny),Value:fmt.Sprintf("%s **Lv. %d**\n❤️ %d/%d\n%s\n%s%s",typeEmoji(pk.Type),pk.Level,hp,battleHP(pk),rarity(pk),teamText,evoText),Inline:true})
	}
	respondEmbed(s,i,&discordgo.MessageEmbed{Title:fmt.Sprintf("📦 Sua Coleção • %d Pokémon",len(p.Pokemon)),Description:"Seu inventário de Pokémon em formato de coleção.\nUse /equipe para escolher os 6 que irão para as batalhas.",Color:0x57F287,Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(p.Pokemon[0].ID)},Fields:fields,Footer:&discordgo.MessageEmbedFooter{Text:"/treinar numero:N • /equipe adicionar • /curar"}})
}

func handleTrain(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := iUser(i).ID
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		respond(s, i, "Informe o número do Pokémon que deseja treinar.")
		return
	}
	index := int(opts[0].IntValue()) - 1
	message, ok := trainPokemon(userID, index)
	respond(s, i, message)
	_ = ok
}

func handleInventory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p, ok := getPlayer(iUser(i).ID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	isca := p.Items["isca"]
	super := p.Items["super_isca"]
	lure := "Nenhuma ativa"
	if !p.ActiveLureUntil.IsZero() && time.Now().Before(p.ActiveLureUntil) {
		lure = fmt.Sprintf("%s — %d min restantes", p.ActiveLureName, int(time.Until(p.ActiveLureUntil).Minutes())+1)
	}
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "🎒 Inventário",
		Description: fmt.Sprintf("💰 **%d Coins**\n\n🎣 **Isca:** %d\n🎣 **Super Isca:** %d\n\n🔥 **Isca ativa:** %s", p.Coins, isca, super, lure),
		Color: 0xF2A900,
	})
}

func handleShop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "🏪 Loja",
		Description: "Use `/comprar` para adquirir um item.",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🎣 Isca", Value: "50 Coins\nDura 10 minutos\n+15% de chance de captura.", Inline: true},
			{Name: "🎣 Super Isca", Value: "100 Coins\nDura 15 minutos\n+25% de chance de captura.", Inline: true},
		},
	})
}

func handleBuy(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		respond(s, i, "Escolha um item para comprar.")
		return
	}
	item := opts[0].StringValue()
	prices := map[string]int{"isca": 50, "super_isca": 100}
	names := map[string]string{"isca": "Isca", "super_isca": "Super Isca"}
	price, exists := prices[item]
	if !exists {
		respond(s, i, "Esse item não existe na loja.")
		return
	}

	userID := iUser(i).ID
	store.mu.Lock()
	p, ok := store.Players[userID]
	if !ok {
		store.mu.Unlock()
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	if p.Coins < price {
		store.mu.Unlock()
		respond(s, i, fmt.Sprintf("❌ Você precisa de **%d Coins**, mas possui **%d**.", price, p.Coins))
		return
	}
	if p.Items == nil {
		p.Items = map[string]int{}
	}
	p.Coins -= price
	p.Items[item]++
	store.Players[userID] = p
	if err := saveStoreLocked(); err != nil {
		log.Printf("save purchase: %v", err)
	}
	store.mu.Unlock()

	respond(s, i, fmt.Sprintf("🛍️ Você comprou **%s** por **%d Coins**! Use `/inventario` para conferir.", names[item], price))
}

func handleUse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		respond(s, i, "Escolha uma isca para usar.")
		return
	}
	item := opts[0].StringValue()
	duration := 10 * time.Minute
	name := "Isca"
	if item == "super_isca" {
		duration = 15 * time.Minute
		name = "Super Isca"
	}
	userID := iUser(i).ID
	store.mu.Lock()
	p, ok := store.Players[userID]
	if !ok {
		store.mu.Unlock()
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	if p.Items == nil || p.Items[item] <= 0 {
		store.mu.Unlock()
		respond(s, i, fmt.Sprintf("❌ Você não possui **%s**. Use `/loja` para comprar.", name))
		return
	}
	p.Items[item]--
	p.ActiveLureName = name
	p.ActiveLureUntil = time.Now().Add(duration)
	store.Players[userID] = p
	if err := saveStoreLocked(); err != nil {
		log.Printf("save lure: %v", err)
	}
	store.mu.Unlock()

	respond(s, i, fmt.Sprintf("🎣 **%s ativada!** Ela ficará ativa por %d minutos e aumentará sua chance de captura. Use `/procurar`.", name, int(duration.Minutes())))
}

func handlePokedex(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := iUser(i).ID
	player, ok := getPlayer(userID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}

	caught := make(map[int]bool)
	for _, pk := range player.Pokemon {
		caught[pk.ID] = true
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(routes))
	for _, route := range routes {
		unique := make(map[int]bool)
		captured := 0
		for _, id := range route.PokemonIDs {
			if unique[id] {
				continue
			}
			unique[id] = true
			if caught[id] {
				captured++
			}
		}

		lines := make([]string, 0, len(unique))
		for _, id := range route.PokemonIDs {
			if !unique[id] {
				continue
			}
			pk := pokemonByID(id)
			if caught[id] {
				lines = append(lines, fmt.Sprintf("✅ **%s**", pk.Name))
			} else {
				lines = append(lines, fmt.Sprintf("⬜ %s", pk.Name))
			}
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name: route.Name,
			Value: fmt.Sprintf("**%d/%d capturados**\n%s", captured, len(unique), strings.Join(lines, " • ")),
		})
	}

	uniqueAll := make(map[int]bool)
	for _, pk := range player.Pokemon {
		uniqueAll[pk.ID] = true
	}

	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "📖 Pokédex",
		Description: fmt.Sprintf("**%d espécies descobertas** • **%d Pokémon capturados**\nComplete as rotas para preencher sua Pokédex.", len(uniqueAll), len(player.Pokemon)),
		Color: 0x5865F2,
		Fields: fields,
	})
}

func coliseumRank(wins int) (string, int, int) {
	switch {
	case wins >= 20:
		return "👑 Elite Four", 20, 0
	case wins >= 10:
		return "🥇 Ouro", 10, 20
	case wins >= 5:
		return "🥈 Prata", 5, 10
	default:
		return "🥉 Bronze", 0, 5
	}
}

func handleTeam(s *discordgo.Session, i *discordgo.InteractionCreate) {
	uid := iUser(i).ID
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 || opts[0].Name == "ver" { handleTeamView(s, i); return }
	store.mu.Lock()
	p, ok := store.Players[uid]
	if !ok { store.mu.Unlock(); respond(s, i, "Você ainda não começou. Use /iniciar primeiro."); return }
	if sub := opts[0]; sub.Name == "limpar" {
		p.Team = nil; store.Players[uid] = p; _ = saveStoreLocked(); store.mu.Unlock()
		respond(s, i, "🧹 Equipe limpa! Use /equipe adicionar para montar sua equipe."); return
	}
	sub := opts[0]
	if len(sub.Options) == 0 { store.mu.Unlock(); respond(s, i, "Informe o número do Pokémon."); return }
	idx := int(sub.Options[0].IntValue()) - 1
	if idx < 0 || idx >= len(p.Pokemon) { store.mu.Unlock(); respond(s, i, fmt.Sprintf("❌ Pokémon inválido. Use /pokemon para ver 1 a %d.", len(p.Pokemon))); return }
	pos := -1
	for n, v := range p.Team { if v == idx { pos = n; break } }
	if sub.Name == "adicionar" {
		if pos >= 0 { store.mu.Unlock(); respond(s, i, "⚠️ Esse Pokémon já está na sua equipe."); return }
		if len(p.Team) >= 6 { store.mu.Unlock(); respond(s, i, "❌ Sua equipe já tem 6 Pokémon."); return }
		p.Team = append(p.Team, idx); name := p.Pokemon[idx].Name; count := len(p.Team)
		store.Players[uid] = p; _ = saveStoreLocked(); store.mu.Unlock()
		respond(s, i, fmt.Sprintf("➕ **%s** entrou na equipe! (%d/6)", name, count)); return
	}
	if sub.Name == "remover" {
		if pos < 0 { store.mu.Unlock(); respond(s, i, "⚠️ Esse Pokémon não está na sua equipe."); return }
		p.Team = append(p.Team[:pos], p.Team[pos+1:]...)
		name := p.Pokemon[idx].Name; count := len(p.Team)
		store.Players[uid] = p; _ = saveStoreLocked(); store.mu.Unlock()
		respond(s, i, fmt.Sprintf("➖ **%s** saiu da equipe. (%d/6)", name, count)); return
	}
	store.mu.Unlock(); respond(s, i, "Use /equipe ver, /equipe adicionar, /equipe remover ou /equipe limpar.")
}


func handleTeamView(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p,ok:=getPlayer(iUser(i).ID);if !ok{respond(s,i,"Você ainda não começou. Use /iniciar primeiro.");return}
	fields:=make([]*discordgo.MessageEmbedField,0,6)
	for slot:=0;slot<6;slot++{
		if slot>=len(p.Team){fields=append(fields,&discordgo.MessageEmbedField{Name:fmt.Sprintf("Slot %d",slot+1),Value:"🔒 **Slot vazio**\nUse /equipe adicionar",Inline:true});continue}
		idx:=p.Team[slot];if idx<0||idx>=len(p.Pokemon){fields=append(fields,&discordgo.MessageEmbedField{Name:fmt.Sprintf("Slot %d",slot+1),Value:"⚠️ Slot inválido",Inline:true});continue}
		pk:=p.Pokemon[idx];pk.Type=pokemonType(pk.ID);hp:=pk.HP;if hp<=0||hp>battleHP(pk){hp=battleHP(pk)}
		fields=append(fields,&discordgo.MessageEmbedField{Name:fmt.Sprintf("🔹 %d. %s",slot+1,pk.Name),Value:fmt.Sprintf("%s **Lv. %d**\n❤️ %d/%d\n%s\n%s",typeEmoji(pk.Type),pk.Level,hp,battleHP(pk),progressBar(hp,battleHP(pk),8),rarity(pk)),Inline:true})
	}
	firstID:=25;if len(p.Team)>0&&p.Team[0]>=0&&p.Team[0]<len(p.Pokemon){firstID=p.Pokemon[p.Team[0]].ID}
	respondEmbed(s,i,&discordgo.MessageEmbed{Title:fmt.Sprintf("👥 Sua equipe (%d/6)",len(p.Team)),Description:"Sua equipe de batalha.\nCada slot mostra nível, HP e raridade.",Color:0xF1C40F,Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(firstID)},Fields:fields,Footer:&discordgo.MessageEmbedFooter{Text:"/equipe adicionar • /equipe remover • /equipe limpar"}})
}

func handleHeal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := iUser(i).ID
	const cost = 20

	store.mu.Lock()
	p, ok := store.Players[userID]
	if !ok {
		store.mu.Unlock()
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	if _, active := store.Battles[userID]; active {
		store.mu.Unlock()
		respond(s, i, "⚔️ Termine sua batalha antes de curar seus Pokémon.")
		return
	}
	if p.Coins < cost {
		store.mu.Unlock()
		respond(s, i, fmt.Sprintf("❌ Curar custa **%d Coins**. Você possui **%d**.", cost, p.Coins))
		return
	}
	p.Coins -= cost
	for idx := range p.Pokemon {
		p.Pokemon[idx].HP = battleHP(p.Pokemon[idx])
	}
	store.Players[userID] = p
	if err := saveStoreLocked(); err != nil {
		log.Printf("save heal: %v", err)
	}
	store.mu.Unlock()

	respond(s, i, fmt.Sprintf("💚 **Todos os seus Pokémon foram curados!**\n\n💰 -%d Coins\n💰 Saldo: %d Coins", cost, p.Coins))
}

func handleLeague(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p, ok := getPlayer(iUser(i).ID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	rank, current, next := coliseumRank(p.ColiseumWins)
	progress := fmt.Sprintf("%d vitórias", p.ColiseumWins)
	if next > 0 {
		progress += fmt.Sprintf("\nFaltam **%d** vitórias para subir.", next-p.ColiseumWins)
	} else {
		progress += "\nVocê alcançou o topo do Coliseu! 👑"
	}
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "🏆 Liga do Coliseu",
		Description: fmt.Sprintf("**%s**\n\n%s", rank, progress),
		Color: 0xF2A900,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🥉 Bronze", Value: "0–4 vitórias", Inline: true},
			{Name: "🥈 Prata", Value: "5–9 vitórias", Inline: true},
			{Name: "🥇 Ouro", Value: "10–19 vitórias", Inline: true},
			{Name: "👑 Elite Four", Value: "20+ vitórias", Inline: true},
			{Name: "Próximo marco", Value: func() string { if next == 0 { return "Topo alcançado" }; return strconv.Itoa(next) + " vitórias" }(), Inline: true},
			{Name: "Progresso", Value: fmt.Sprintf("%d", p.ColiseumWins-current), Inline: true},
		},
	})
}

func handleRanking(s *discordgo.Session, i *discordgo.InteractionCreate) {
	store.mu.RLock()
	players := make([]Player, 0, len(store.Players))
	for _, p := range store.Players {
		players = append(players, p)
	}
	store.mu.RUnlock()

	sort.Slice(players, func(i, j int) bool {
		if players[i].ColiseumWins == players[j].ColiseumWins {
			return players[i].Level > players[j].Level
		}
		return players[i].ColiseumWins > players[j].ColiseumWins
	})

	limit := len(players)
	if limit > 10 { limit = 10 }
	lines := make([]string, 0, limit)
	for idx := 0; idx < limit; idx++ {
		lines = append(lines, fmt.Sprintf("**%d. %s** — %d vitórias — Lv. %d", idx+1, players[idx].Username, players[idx].ColiseumWins, players[idx].Level))
	}
	if len(lines) == 0 {
		lines = append(lines, "Nenhum treinador começou a jornada ainda.")
	}
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "🏆 Ranking de Treinadores",
		Description: strings.Join(lines, "\n"),
		Color: 0x5865F2,
	})
}


func findGym(id string) *Gym {
	for n := range gyms {
		if gyms[n].ID == id {
			return &gyms[n]
		}
	}
	return nil
}



func handleGyms(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p,ok:=getPlayer(iUser(i).ID)
	if !ok{respond(s,i,"Você ainda não começou. Use /iniciar primeiro.");return}
	embeds:=make([]*discordgo.MessageEmbed,0,len(gyms)+1)
	embeds=append(embeds,&discordgo.MessageEmbed{
		Title:"🏟️ Ginásios de Kanto",
		Description:fmt.Sprintf("Desafie os líderes, conquiste as **8 insígnias** e torne-se o Campeão!\n\n**Progresso:** %d/8  %s",len(p.Badges),progressBar(len(p.Badges),8,8)),
		Color:0xF1C40F,
		Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(112)},
		Footer:&discordgo.MessageEmbedFooter{Text:"Use /ginasio numero:N • A ordem dos ginásios é obrigatória"},
	})
	for n,g:=range gyms{
		status:="🔒 Bloqueado";if p.Level>=g.UnlockLevel{status="⚔️ Disponível"}
		if n>0{hasPrev:=false;for _,b:=range p.Badges{if b==gyms[n-1].Badge{hasPrev=true;break}};if !hasPrev{status="🔒 Requer a anterior"}}
		for _,b:=range p.Badges{if b==g.Badge{status="✅ Conquistado"}}
		ace:=gymAce(g)
		embeds=append(embeds,&discordgo.MessageEmbed{
			Title:fmt.Sprintf("%d. %s",n+1,g.Name),
			Description:fmt.Sprintf("👤 **Líder:** %s\n%s **Tipo:** %s\n🎖️ **%s**\n\n**%s**",g.Leader,typeEmoji(g.Type),g.Type,g.Badge,status),
			Color:typeColor(g.Type),
			Thumbnail:&discordgo.MessageEmbedThumbnail{URL:trainerSpriteURL(g.Leader)},
			Image:&discordgo.MessageEmbedImage{URL:officialArtworkURL(ace.ID)},
			Fields:[]*discordgo.MessageEmbedField{
				{Name:"🔓 Requisito",Value:fmt.Sprintf("Level %d",g.UnlockLevel),Inline:true},
				{Name:"⭐ Ace",Value:fmt.Sprintf("%s · Lv. %d",ace.Name,ace.Level),Inline:true},
				{Name:"💰 Recompensa",Value:fmt.Sprintf("%d Coins + XP",g.RewardCoins),Inline:true},
			},
			Footer:&discordgo.MessageEmbedFooter{Text:"Use /ginasio numero:"+strconv.Itoa(n+1)+" para desafiar"},
		})
	}
	respondMultiEmbeds(s,i,embeds,nil)
}

func handleGymBattle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts)==0 { respond(s,i,"Use /ginasios para ver os ginásios."); return }
	n := int(opts[0].IntValue())-1
	if n<0 || n>=len(gyms) { respond(s,i,"❌ Ginásio inválido. Use /ginasios."); return }
	g := gyms[n]; uid := iUser(i).ID
	store.mu.Lock(); p,ok := store.Players[uid]
	if !ok { store.mu.Unlock(); respond(s,i,"Você ainda não começou. Use /iniciar primeiro."); return }
	if p.Level<g.UnlockLevel { store.mu.Unlock(); respond(s,i,fmt.Sprintf("🔒 Você precisa estar no nível %d.",g.UnlockLevel)); return }
	if n > 0 {
		previousBadge := gyms[n-1].Badge
		hasPrevious := false
		for _,b := range p.Badges {
			if b == previousBadge {
				hasPrevious = true
				break
			}
		}
		if !hasPrevious {
			store.mu.Unlock()
			respond(s,i,fmt.Sprintf("🔒 Você precisa conquistar **%s** antes de desafiar este ginásio.", previousBadge))
			return
		}
	}
	for _,b := range p.Badges { if b==g.Badge { store.mu.Unlock(); respond(s,i,"✅ Você já conquistou essa insígnia."); return } }
	if _,active:=store.Battles[uid]; active { store.mu.Unlock(); respond(s,i,"⚔️ Você já está em uma batalha."); return }
	if len(p.Team)==0 { store.mu.Unlock(); respond(s,i,"❌ Monte sua equipe com /equipe."); return }
	idx:=p.Team[0]; if idx<0 || idx>=len(p.Pokemon) { store.mu.Unlock(); respond(s,i,"❌ Sua equipe está inválida."); return }
	chosen:=p.Pokemon[idx]; chosen.Type=pokemonType(chosen.ID); hp:=chosen.HP; if hp<=0 || hp>battleHP(chosen) { hp=battleHP(chosen) }
	op:=g.Pokemon[rand.Intn(len(g.Pokemon))]; op.Type=pokemonType(op.ID)
	battle:=Battle{OwnerID:uid,PlayerPokemon:chosen,Opponent:op,PlayerHP:hp,OpponentHP:battleHP(op),GymID:g.ID, Elite:false, EliteIndex:-1}
	store.Battles[uid]=battle; store.mu.Unlock()
	respondGymBattle(s,i,battle,g)
}

func handleColiseum(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := iUser(i).ID
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		respond(s, i, "Escolha o número do Pokémon que deseja usar no Coliseu.")
		return
	}
	index := int(opts[0].IntValue()) - 1

	store.mu.Lock()
	player, ok := store.Players[userID]
	if !ok {
		store.mu.Unlock()
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	if index < 0 || index >= len(player.Pokemon) {
		store.mu.Unlock()
		respond(s, i, fmt.Sprintf("❌ Pokémon inválido. Use /pokemon para ver os números de 1 a %d.", len(player.Pokemon)))
		return
	}
	if _, active := store.Battles[userID]; active {
		store.mu.Unlock()
		respond(s, i, "⚔️ Você já está em uma batalha! Termine-a antes de entrar em outra.")
		return
	}

	if len(player.Team) == 0 { store.mu.Unlock(); respond(s, i, "❌ Sua equipe está vazia. Use /equipe adicionar primeiro."); return }
	inTeam := false
	for _, teamIndex := range player.Team { if teamIndex == index { inTeam = true; break } }
	if !inTeam { store.mu.Unlock(); respond(s, i, "❌ Esse Pokémon não está na sua equipe. Use /equipe ver ou /equipe adicionar."); return }
	chosen := player.Pokemon[index]
	chosen.Type = pokemonType(chosen.ID)
	maxHP := battleHP(chosen)
	currentHP := chosen.HP
	if currentHP <= 0 || currentHP > maxHP {
		currentHP = maxHP
	}
	opponent := randomOpponent(player.Level)
	battle := Battle{
		OwnerID: userID,
		PlayerPokemon: chosen,
		Opponent: opponent,
		PlayerHP: currentHP,
		OpponentHP: battleHP(opponent),
		Elite: false,
		EliteIndex: -1,
	}
	store.Battles[userID] = battle
	store.mu.Unlock()

	respondBattle(s, i, battle)
}

func startEliteBattle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID:=iUser(i).ID
	store.mu.Lock()
	p,ok:=store.Players[userID]
	if !ok { store.mu.Unlock(); editComponent(s,i,"Você ainda não começou. Use /iniciar."); return }
	if len(p.Badges)<8 { store.mu.Unlock(); editComponent(s,i,"🔒 Você precisa das **8 insígnias** para entrar na Liga Pokémon."); return }
	if p.Champion { store.mu.Unlock(); editComponent(s,i,"👑 Você já é o Campeão!"); return }
	if _,active:=store.Battles[userID]; active { store.mu.Unlock(); editComponent(s,i,"⚔️ Você já está em uma batalha."); return }
	if len(p.Team)==0 { store.mu.Unlock(); editComponent(s,i,"❌ Monte sua equipe com /equipe."); return }
	battle:=createLeagueBattle(p,p.LeagueWins)
	store.Battles[userID]=battle
	store.mu.Unlock()
	respondLeagueBattle(s,i,battle,battle.EliteIndex)
}

func createLeagueBattle(p Player, stage int) Battle {
	teamIndex := p.Team[0]
	chosen := p.Pokemon[teamIndex]
	chosen.Type = pokemonType(chosen.ID)
	opponents := leagueTrainers[stage].Pokemon
	op := opponents[rand.Intn(len(opponents))]
	op.Type = pokemonType(op.ID)
	return Battle{
		OwnerID:p.DiscordID, PlayerPokemon:chosen, Opponent:op,
		PlayerHP:battleHP(chosen), OpponentHP:battleHP(op),
		Elite:true, EliteIndex:stage,
	}
}



func handleElite(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p,ok:=getPlayer(iUser(i).ID)
	if !ok{respond(s,i,"Você ainda não começou. Use /iniciar primeiro.");return}
	embeds:=make([]*discordgo.MessageEmbed,0,len(leagueTrainers)+1)
	description:="Derrote os 4 membros da Elite Four em sequência e enfrente o Campeão."
	if len(p.Badges)<8{description+=fmt.Sprintf("\n\n🔒 Requer **8 insígnias**\n%s\n**%d/8**",progressBar(len(p.Badges),8,8),len(p.Badges))
	}else if p.Champion{description+="\n\n👑 **VOCÊ É O CAMPEÃO!**\nHall da Fama desbloqueado."
	}else{description+=fmt.Sprintf("\n\n⚔️ **Progresso %d/5**\n%s",p.LeagueWins,progressBar(p.LeagueWins,5,10))}
	components:=[]discordgo.MessageComponent{}
	if len(p.Badges)>=8&&!p.Champion&&p.LeagueWins<5{components=[]discordgo.MessageComponent{discordgo.ActionsRow{Components:[]discordgo.MessageComponent{discordgo.Button{CustomID:"elite:start:"+p.DiscordID,Label:"🏆 Iniciar Desafio da Elite Four",Style:discordgo.SuccessButton}}}}}
	embeds=append(embeds,&discordgo.MessageEmbed{Title:"🏆 Elite Four • Liga Pokémon",Description:description,Color:0x9B59B6,Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(149)},Footer:&discordgo.MessageEmbedFooter{Text:"Lorelei • Bruno • Agatha • Lance • Campeão Blue"}})
	for idx,trainer:=range leagueTrainers{
		levels:=make([]string,0,len(trainer.Pokemon));for _,pk:=range trainer.Pokemon{levels=append(levels,fmt.Sprintf("%s Lv.%d",pk.Name,pk.Level))}
		ace:=trainer.Pokemon[len(trainer.Pokemon)-1]
		embedColor:=typeColor(trainer.Type);if trainer.IsChampion{embedColor=0xF1C40F}
		embeds=append(embeds,&discordgo.MessageEmbed{
			Title:fmt.Sprintf("%s %d. %s",func()string{if trainer.IsChampion{return "👑"};return "🛡️"}(),idx+1,trainer.Name),
			Description:fmt.Sprintf("%s **%s**\n%s\n**%s**",typeEmoji(trainer.Type),trainer.Type,strings.Join(levels," • "),leagueStatus(p,idx)),
			Color:embedColor,
			Thumbnail:&discordgo.MessageEmbedThumbnail{URL:trainerSpriteURL(trainer.Name)},
			Image:&discordgo.MessageEmbedImage{URL:officialArtworkURL(ace.ID)},
			Fields:[]*discordgo.MessageEmbedField{
				{Name:"👥 Equipe",Value:fmt.Sprintf("%d Pokémon",len(trainer.Pokemon)),Inline:true},
				{Name:"⭐ Ace",Value:fmt.Sprintf("%s · Lv. %d",ace.Name,ace.Level),Inline:true},
				{Name:"🎯 Etapa",Value:fmt.Sprintf("%d/5",idx+1),Inline:true},
			},
		})
	}
	respondMultiEmbeds(s,i,embeds,components)
}

func handleBattleAttack(s *discordgo.Session, i *discordgo.InteractionCreate, userID string, moveIndex int) {
	store.mu.Lock()
	battle, ok := store.Battles[userID]
	if !ok {
		store.mu.Unlock()
		editComponent(s, i, "❌ Você não está em uma batalha ativa.")
		return
	}

	moves := movesFor(battle.PlayerPokemon)
	if moveIndex < 0 || moveIndex >= len(moves) {
		store.mu.Unlock()
		editComponent(s, i, "❌ Golpe inválido.")
		return
	}
	move := moves[moveIndex]
	playerDamage, multiplier := calculateDamage(battle.PlayerPokemon, battle.Opponent, move)
	battle.OpponentHP -= playerDamage
	effectText := effectivenessText(multiplier)
	messages := []string{fmt.Sprintf("⚔️ **%s usou %s!** %d de dano%s", battle.PlayerPokemon.Name, move.Name, playerDamage, effectText)}

	if battle.OpponentHP <= 0 {
		p := store.Players[userID]

		if battle.Elite {
			stage := battle.EliteIndex
			rewardCoins := 500 + stage*150
			rewardXP := 120 + stage*60
			p.Coins += rewardCoins
			p.LeagueWins++
			battle.PlayerPokemon.HP = battle.PlayerHP
			p.Pokemon = addPokemonXP(p.Pokemon,battle.PlayerPokemon,rewardXP)

			if stage < len(leagueTrainers)-1 {
				next := stage+1
				nextBattle := createLeagueBattle(p,next)
				store.Players[userID]=p
				store.Battles[userID]=nextBattle
				if err:=saveStoreLocked(); err!=nil { log.Printf("save league progress: %v",err) }
				store.mu.Unlock()
				editLeagueBattle(s,i,nextBattle,next,fmt.Sprintf("🏆 **%s derrotado!** +%d Coins • +%d XP\n\n➡️ Próximo: **%s**",leagueTrainers[stage].Name,rewardCoins,rewardXP,leagueTrainers[next].Name))
				return
			}

			p.Champion=true
			p.Coins+=2000
			p.XP+=500
			for p.XP >= p.Level*50 { p.XP-=p.Level*50; p.Level++ }
			store.Players[userID]=p
			delete(store.Battles,userID)
			if err:=saveStoreLocked(); err!=nil { log.Printf("save championship: %v",err) }
			store.mu.Unlock()
			editComponent(s,i,fmt.Sprintf("👑 **VOCÊ É O CAMPEÃO!**\n\n%s\n\n🏆 Elite Four concluída!\n💰 +%d Coins\n✨ +%d XP\n💎 Bônus de Campeão: +2.000 Coins\n\n**Hall da Fama desbloqueado!**",strings.Join(messages,"\n"),rewardCoins,rewardXP+500))
			return
		}

		rewardCoins := 15 + battle.Opponent.Level*3
		rewardXP := 15 + battle.Opponent.Level*5
		gym := findGym(battle.GymID)
		if gym != nil {
			rewardCoins = gym.RewardCoins
			rewardXP = 25 + gym.UnlockLevel*5
			p.Coins += rewardCoins
			p.Badges = append(p.Badges,gym.Badge)
		} else {
			p.Coins += rewardCoins
			p.ColiseumWins++
		}
		battle.PlayerPokemon.HP=battle.PlayerHP
		p.Pokemon=addPokemonXP(p.Pokemon,battle.PlayerPokemon,rewardXP)
		store.Players[userID]=p
		delete(store.Battles,userID)
		if err:=saveStoreLocked(); err!=nil { log.Printf("save battle victory: %v",err) }
		store.mu.Unlock()
		if gym != nil {
			editComponent(s,i,fmt.Sprintf("🏆 **Ginásio derrotado!**\n\n%s\n\n🏅 **%s** conquistada!\n💰 **+%d Coins**\n✨ **+%d XP** para %s.",strings.Join(messages,"\n"),gym.Badge,rewardCoins,rewardXP,battle.PlayerPokemon.Name))
		} else {
			editComponent(s,i,fmt.Sprintf("🏆 **Vitória!**\n\n%s\n💰 **+%d Coins**\n✨ **+%d XP** para %s.",strings.Join(messages,"\n"),rewardCoins,rewardXP,battle.PlayerPokemon.Name))
		}
		return
	}

	opponentMoves := movesFor(battle.Opponent)
	opponentMove := opponentMoves[rand.Intn(len(opponentMoves))]
	opponentDamage, opponentMultiplier := calculateDamage(battle.Opponent, battle.PlayerPokemon, opponentMove)
	battle.PlayerHP -= opponentDamage
	messages = append(messages, fmt.Sprintf("💥 **%s usou %s!** %d de dano%s", battle.Opponent.Name, opponentMove.Name, opponentDamage, effectivenessText(opponentMultiplier)))

	if battle.PlayerHP <= 0 {
		p := store.Players[userID]
		for idx := range p.Pokemon {
			if p.Pokemon[idx].ID == battle.PlayerPokemon.ID && p.Pokemon[idx].Level == battle.PlayerPokemon.Level && p.Pokemon[idx].Shiny == battle.PlayerPokemon.Shiny {
				p.Pokemon[idx].HP = 0
				break
			}
		}
		store.Players[userID] = p
		if err := saveStoreLocked(); err != nil {
			log.Printf("save battle defeat: %v", err)
		}
		delete(store.Battles, userID)
		store.mu.Unlock()
		editComponent(s, i, fmt.Sprintf("💀 **Derrota!**\n\n%s\nSeu Pokémon ficou sem HP. Use /curar antes de tentar novamente.", strings.Join(messages, "\n")))
		return
	}

	p := store.Players[userID]
	for idx := range p.Pokemon {
		if p.Pokemon[idx].ID == battle.PlayerPokemon.ID && p.Pokemon[idx].Level == battle.PlayerPokemon.Level && p.Pokemon[idx].Shiny == battle.PlayerPokemon.Shiny {
			p.Pokemon[idx].HP = battle.PlayerHP
			break
		}
	}
	store.Players[userID] = p
	store.Battles[userID] = battle
	if err := saveStoreLocked(); err != nil {
		log.Printf("save battle state: %v", err)
	}
	store.mu.Unlock()
	if battle.Elite {
		editLeagueBattle(s,i,battle,battle.EliteIndex,strings.Join(messages,"\n"))
		return
	}
	editBattle(s, i, battle, strings.Join(messages, "\n"))
}



func respondLeagueBattle(s *discordgo.Session, i *discordgo.InteractionCreate, battle Battle, stage int) {
	trainer:=leagueTrainers[stage]
	respondEmbedWithComponents(s,i,&discordgo.MessageEmbed{
		Title:fmt.Sprintf("🏆 %s • Desafio %d/5",trainer.Name,stage+1),
		Description:fmt.Sprintf("**%s** VS **%s**\n%s **Tipo:** %s\n\n⚠️ Sequência obrigatória da Liga Pokémon.",battle.PlayerPokemon.Name,battle.Opponent.Name,typeEmoji(trainer.Type),trainer.Type),
		Color:typeColor(trainer.Type),
		Thumbnail:&discordgo.MessageEmbedThumbnail{URL:trainerSpriteURL(trainer.Name)},
		Image:&discordgo.MessageEmbedImage{URL:officialArtworkURL(battle.PlayerPokemon.ID)},
		Fields:battleFields(battle),
	},battleComponents(battle))
}

func editLeagueBattle(s *discordgo.Session, i *discordgo.InteractionCreate, battle Battle, stage int, logText string) {
	trainer:=leagueTrainers[stage]
	content:=logText
	components:=battleComponents(battle)
	_,err:=s.InteractionResponseEdit(i.Interaction,&discordgo.WebhookEdit{
		Content:&content,
		Embeds:&[]*discordgo.MessageEmbed{{
			Title:fmt.Sprintf("🏆 %s • Desafio %d/5",trainer.Name,stage+1),
			Description:fmt.Sprintf("**%s** enfrenta **%s**!",battle.PlayerPokemon.Name,battle.Opponent.Name),
			Color:0x9B59B6,
			Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(battle.Opponent.ID)},
			Image:&discordgo.MessageEmbedImage{URL:officialArtworkURL(battle.PlayerPokemon.ID)},
			Fields:battleFields(battle),
		}},
		Components:&components,
	})
	if err!=nil { log.Printf("edit league battle: %v",err) }
}


func battleFields(battle Battle) []*discordgo.MessageEmbedField {
	moves:=movesFor(battle.PlayerPokemon);pm:=battleHP(battle.PlayerPokemon);om:=battleHP(battle.Opponent)
	return []*discordgo.MessageEmbedField{
		{Name:"🔥 "+battle.PlayerPokemon.Name,Value:fmt.Sprintf("Lv. %d\n%s\n❤️ **%d/%d HP**",battle.PlayerPokemon.Level,progressBar(battle.PlayerHP,pm,10),battle.PlayerHP,pm),Inline:true},
		{Name:"👾 "+battle.Opponent.Name,Value:fmt.Sprintf("Lv. %d\n%s\n❤️ **%d/%d HP**",battle.Opponent.Level,progressBar(battle.OpponentHP,om,10),battle.OpponentHP,om),Inline:true},
		{Name:"🎯 Ataques",Value:fmt.Sprintf("1️⃣ **%s** · %s\n2️⃣ **%s** · %s\n3️⃣ **%s** · %s",moves[0].Name,moves[0].Type,moves[1].Name,moves[1].Type,moves[2].Name,moves[2].Type)},
	}
}

func battleComponents(battle Battle) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: "golpe0:" + battle.OwnerID, Label: "1️⃣ " + movesFor(battle.PlayerPokemon)[0].Name, Style: discordgo.DangerButton},
		discordgo.Button{CustomID: "golpe1:" + battle.OwnerID, Label: "2️⃣ " + movesFor(battle.PlayerPokemon)[1].Name, Style: discordgo.PrimaryButton},
		discordgo.Button{CustomID: "golpe2:" + battle.OwnerID, Label: "3️⃣ " + movesFor(battle.PlayerPokemon)[2].Name, Style: discordgo.SuccessButton},
	}}}
}

func respondBattle(s *discordgo.Session, i *discordgo.InteractionCreate, battle Battle) {
	title := "⚔️ Coliseu"; color := 0xED4245
	if battle.GymID != "" { title = "🏟️ Ginásio"; color = 0xF1C40F }
	respondEmbedWithComponents(s,i,&discordgo.MessageEmbed{
		Title:title,
		Description:fmt.Sprintf("**%s** enfrenta **%s**!",battle.PlayerPokemon.Name,battle.Opponent.Name),
		Color:color,
		Thumbnail:&discordgo.MessageEmbedThumbnail{URL:officialArtworkURL(battle.Opponent.ID)},
		Image:&discordgo.MessageEmbedImage{URL:officialArtworkURL(battle.PlayerPokemon.ID)},
		Fields:battleFields(battle),
	},battleComponents(battle))
}



func respondGymBattle(s *discordgo.Session, i *discordgo.InteractionCreate, battle Battle, gym Gym) {
	respondEmbedWithComponents(s,i,&discordgo.MessageEmbed{
		Title:"🏟️ "+gym.Name+" • VS "+gym.Leader,
		Description:fmt.Sprintf("👤 **Líder:** %s\n%s **Tipo:** %s\n\n**%s** VS **%s**",gym.Leader,typeEmoji(gym.Type),gym.Type,battle.PlayerPokemon.Name,battle.Opponent.Name),
		Color:typeColor(gym.Type),
		Thumbnail:&discordgo.MessageEmbedThumbnail{URL:trainerSpriteURL(gym.Leader)},
		Image:&discordgo.MessageEmbedImage{URL:officialArtworkURL(battle.PlayerPokemon.ID)},
		Fields:battleFields(battle),
	},battleComponents(battle))
}

func editBattle(s *discordgo.Session, i *discordgo.InteractionCreate, battle Battle, logText string) {
	content := logText
	components := battleComponents(battle)
	title := "⚔️ Coliseu"
	color := 0xED4245
	if battle.GymID != "" { title = "🏟️ Ginásio"; color = 0xF1C40F }
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
		Embeds: &[]*discordgo.MessageEmbed{{
			Title: title,
			Description: fmt.Sprintf("**%s** vs **%s**", battle.PlayerPokemon.Name, battle.Opponent.Name),
			Color: color,
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: officialArtworkURL(battle.Opponent.ID)},
			Image: &discordgo.MessageEmbedImage{URL: officialArtworkURL(battle.PlayerPokemon.ID)},
			Fields: battleFields(battle),
		}},
		Components: &components,
	})
	if err != nil {
		log.Printf("edit battle: %v", err)
	}
}

func randomOpponent(trainerLevel int) Pokemon {
	candidates := pokemonPool
	pk := candidates[rand.Intn(len(candidates))]
	minLevel := trainerLevel
	if minLevel < 1 {
		minLevel = 1
	}
	maxLevel := minLevel + 2
	pk.Level = minLevel + rand.Intn(maxLevel-minLevel+1)
	pk.XP = 0
	pk.XPToNext = xpToNext(pk.Level)
	pk.Shiny = false
	pk.Type = pokemonType(pk.ID)
	return pk
}

func battleHP(pk Pokemon) int {
	return 30 + pk.Level*10
}

func battleDamage(pk Pokemon) int {
	return 5 + pk.Level*2 + rand.Intn(6)
}

func pokemonType(id int) string {
	types := map[int]string{
		1: "Planta", 2: "Planta", 3: "Planta",
		4: "Fogo", 5: "Fogo", 6: "Fogo",
		7: "Água", 8: "Água", 9: "Água",
		10: "Inseto", 11: "Inseto", 12: "Inseto",
		13: "Inseto", 14: "Inseto", 15: "Inseto",
		16: "Normal/Voador", 17: "Normal/Voador", 18: "Normal/Voador",
		19: "Normal", 20: "Normal", 21: "Normal/Voador", 22: "Normal/Voador",
		25: "Elétrico", 26: "Elétrico",
		27: "Terra", 31: "Terra", 34: "Veneno",
		29: "Veneno", 30: "Veneno", 32: "Veneno", 33: "Veneno",
		35: "Fada",
		41: "Veneno/Voador", 42: "Veneno/Voador",
		43: "Planta/Veneno", 44: "Planta/Veneno", 45: "Planta/Veneno",
		52: "Normal", 53: "Normal",
		58: "Fogo", 64: "Psíquico", 65: "Psíquico",
		70: "Planta/Veneno", 78: "Fogo",
		68: "Lutador", 80: "Água/Psíquico", 87: "Água/Gelo", 91: "Água/Gelo",
		89: "Veneno", 93: "Fantasma/Veneno", 94: "Fantasma/Veneno", 95: "Pedra/Terra",
		106: "Lutador", 107: "Lutador", 124: "Gelo/Psíquico", 130: "Água/Voador",
		142: "Pedra/Voador", 148: "Dragão", 149: "Dragão/Voador", 100: "Elétrico",
		109: "Veneno", 110: "Veneno", 111: "Terra/Pedra",
		112: "Terra/Pedra", 114: "Planta", 120: "Água", 121: "Água", 122: "Psíquico/Fada",
		144: "Gelo/Voador", 145: "Elétrico/Voador", 146: "Fogo/Voador",
		150: "Psíquico", 151: "Psíquico",
		74: "Pedra/Terra", 75: "Pedra/Terra",
		102: "Planta/Psíquico", 123: "Inseto/Voador", 128: "Normal",
	}
	if t, ok := types[id]; ok {
		return t
	}
	return "Normal"
}

func movesFor(pk Pokemon) []Move {
	switch pk.Type {
	case "Fogo":
		return []Move{{"Arranhão", "Normal", 40}, {"Brasa", "Fogo", 45}, {"Corte de Fogo", "Fogo", 60}}
	case "Água":
		return []Move{{"Investida", "Normal", 40}, {"Jato d'Água", "Água", 45}, {"Pulso d'Água", "Água", 60}}
	case "Planta", "Planta/Veneno":
		return []Move{{"Investida", "Normal", 40}, {"Chicote de Vinha", "Planta", 45}, {"Folha Navalha", "Planta", 60}}
	case "Elétrico":
		return []Move{{"Investida", "Normal", 40}, {"Choque do Trovão", "Elétrico", 45}, {"Faísca", "Elétrico", 60}}
	case "Inseto", "Inseto/Voador":
		return []Move{{"Investida", "Normal", 40}, {"Picada", "Inseto", 45}, {"Corte Furioso", "Inseto", 60}}
	case "Voador", "Normal/Voador", "Veneno/Voador":
		return []Move{{"Investida", "Normal", 40}, {"Rajada", "Voador", 45}, {"Ataque de Asa", "Voador", 60}}
	case "Pedra/Terra":
		return []Move{{"Investida", "Normal", 40}, {"Arremesso de Pedra", "Pedra", 45}, {"Deslizamento", "Pedra", 60}}
	case "Água/Gelo":
		return []Move{{"Investida", "Normal", 40}, {"Jato d'Água", "Água", 45}, {"Raio de Gelo", "Gelo", 60}}
	case "Gelo/Psíquico":
		return []Move{{"Confusão", "Psíquico", 40}, {"Raio de Gelo", "Gelo", 45}, {"Psíquico", "Psíquico", 60}}
	case "Fantasma/Veneno":
		return []Move{{"Lambida", "Fantasma", 40}, {"Bola Sombria", "Fantasma", 45}, {"Onda Venenosa", "Veneno", 60}}
	case "Lutador":
		return []Move{{"Investida", "Normal", 40}, {"Golpe Karate", "Lutador", 45}, {"Chute", "Lutador", 60}}
	case "Água/Voador":
		return []Move{{"Investida", "Normal", 40}, {"Jato d'Água", "Água", 45}, {"Ataque de Asa", "Voador", 60}}
	case "Pedra/Voador":
		return []Move{{"Investida", "Normal", 40}, {"Arremesso de Pedra", "Pedra", 45}, {"Ataque de Asa", "Voador", 60}}
	case "Dragão":
		return []Move{{"Investida", "Normal", 40}, {"Garra de Dragão", "Dragão", 45}, {"Fúria do Dragão", "Dragão", 60}}
	case "Dragão/Voador":
		return []Move{{"Investida", "Normal", 40}, {"Garra de Dragão", "Dragão", 45}, {"Ataque de Asa", "Voador", 60}}
	case "Fantasma":
		return []Move{{"Lambida", "Fantasma", 40}, {"Bola Sombria", "Fantasma", 45}, {"Maldição", "Fantasma", 60}}
	case "Veneno":
		return []Move{{"Investida", "Normal", 40}, {"Picada Venenosa", "Veneno", 45}, {"Ácido", "Veneno", 60}}
	case "Fada":
		return []Move{{"Tapa", "Normal", 40}, {"Vento de Fada", "Fada", 45}, {"Brilho Mágico", "Fada", 60}}
	case "Planta/Psíquico":
		return []Move{{"Confusão", "Psíquico", 40}, {"Absorver", "Planta", 45}, {"Psíquico", "Psíquico", 60}}
	default:
		return []Move{{"Investida", "Normal", 40}, {"Ataque Rápido", "Normal", 45}, {"Golpe Forte", "Normal", 60}}
	}
}

func calculateDamage(attacker Pokemon, defender Pokemon, move Move) (int, float64) {
	base := float64(move.Power) + float64(attacker.Level*2)
	multiplier := typeMultiplier(move.Type, defender.Type)
	damage := int(base/10*multiplier) + rand.Intn(6)
	if damage < 1 {
		damage = 1
	}
	return damage, multiplier
}

func typeMultiplier(moveType, defenderType string) float64 {
	// Regras principais do ciclo e algumas relações clássicas.
	parts := strings.Split(defenderType, "/")
	multiplier := 1.0
	for _, defender := range parts {
		m := 1.0
		switch moveType {
		case "Fogo":
			if defender == "Planta" || defender == "Inseto" { m = 2 }
			if defender == "Água" || defender == "Pedra" { m = 0.5 }
		case "Água":
			if defender == "Fogo" || defender == "Pedra" { m = 2 }
			if defender == "Planta" { m = 0.5 }
		case "Planta":
			if defender == "Água" || defender == "Pedra" { m = 2 }
			if defender == "Fogo" || defender == "Inseto" { m = 0.5 }
		case "Elétrico":
			if defender == "Água" || defender == "Voador" { m = 2 }
			if defender == "Planta" { m = 0.5 }
		case "Inseto":
			if defender == "Planta" || defender == "Psíquico" { m = 2 }
			if defender == "Fogo" || defender == "Pedra" { m = 0.5 }
		case "Pedra":
			if defender == "Fogo" || defender == "Inseto" || defender == "Voador" || defender == "Gelo" { m = 2 }
		case "Gelo":
			if defender == "Dragão" || defender == "Voador" { m = 2 }
		case "Fantasma":
			if defender == "Psíquico" || defender == "Fantasma" { m = 2 }
		case "Lutador":
			if defender == "Normal" || defender == "Pedra" || defender == "Gelo" { m = 2 }
		case "Dragão":
			if defender == "Dragão" { m = 2 }
		case "Veneno":
			if defender == "Planta" || defender == "Fada" { m = 2 }
		case "Fada":
			if defender == "Veneno" { m = 0.5 }
		case "Psíquico":
			if defender == "Veneno" { m = 2 }
		}
		multiplier *= m
	}
	return multiplier
}

func effectivenessText(multiplier float64) string {
	if multiplier >= 2 {
		return " — **Super eficaz!** 💥"
	}
	if multiplier > 0 && multiplier < 1 {
		return " — **Pouco eficaz...**"
	}
	return ""
}

func addPokemonXP(pokemon []Pokemon, target Pokemon, amount int) []Pokemon {
	for idx := range pokemon {
		if pokemon[idx].ID == target.ID && pokemon[idx].Level == target.Level && pokemon[idx].Shiny == target.Shiny {
			pokemon[idx].XP += amount
			if pokemon[idx].XPToNext <= 0 {
				pokemon[idx].XPToNext = xpToNext(pokemon[idx].Level)
			}
			for pokemon[idx].XP >= pokemon[idx].XPToNext {
				pokemon[idx].XP -= pokemon[idx].XPToNext
				pokemon[idx].Level++
				pokemon[idx].XPToNext = xpToNext(pokemon[idx].Level)
				if evo := evolutionFor(pokemon[idx]); evo != nil && pokemon[idx].Level >= evo.Level {
					pokemon[idx].ID = evo.ToID
					pokemon[idx].Name = evo.ToName
				}
			}
			break
		}
	}
	return pokemon
}

func handleHunt(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := iUser(i).ID
	store.mu.Lock()
	player, ok := store.Players[userID]
	if !ok {
		store.mu.Unlock()
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}
	if e, ok := store.Encounters[userID]; ok && time.Now().Before(e.ExpiresAt) {
		store.mu.Unlock()
		respond(s, i, "Você já está em um encontro! Use /capturar ou /fugir.")
		return
	}

	route := getRoute(player.CurrentRoute)
	if route == nil {
		player.CurrentRoute = "route1"
		route = getRoute("route1")
		store.Players[userID] = player
	}
	if player.Level < route.UnlockLevel {
		store.mu.Unlock()
		respond(s, i, fmt.Sprintf("🔒 **%s** desbloqueia no Level %d.", route.Name, route.UnlockLevel))
		return
	}

	pk, special := legendaryEncounter(player)
	if !special {
		pk = randomPokemonForRoute(*route, player.Pokemon)
	}
	lureText := "Nenhuma"
	if !player.ActiveLureUntil.IsZero() && time.Now().Before(player.ActiveLureUntil) {
		lureText = fmt.Sprintf("%s (%d min restantes)", player.ActiveLureName, int(time.Until(player.ActiveLureUntil).Minutes())+1)
	}
	store.Encounters[userID] = Encounter{OwnerID: userID, Pokemon: pk, ExpiresAt: time.Now().Add(60 * time.Second)}
	store.mu.Unlock()

	title := "🌿 Pokémon selvagem apareceu!"
	description := fmt.Sprintf("Um **%s** selvagem apareceu na **%s**!", pk.Name, route.Name)
	color := 0xFEE75C
	if special {
		title = "🌟 ENCONTRO LENDÁRIO!"
		description = fmt.Sprintf("Uma energia lendária tomou conta da **%s**! **%s** apareceu!", route.Name, pk.Name)
		color = 0x9B59B6
	}
	respondEmbedWithComponents(s, i, &discordgo.MessageEmbed{
		Title: title,
		Description: description,
		Color: color,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: spriteURL(pk.ID)},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Level", Value: strconv.Itoa(pk.Level), Inline: true},
			{Name: "Raridade", Value: rarity(pk), Inline: true},
			{Name: "🗺️ Rota", Value: route.Name, Inline: true},
			{Name: "⏱️ Encontro", Value: "60 segundos", Inline: true},
			{Name: "🎣 Isca", Value: lureText, Inline: true},
		},
	}, []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: "capturar:" + userID, Label: "🎯 Capturar", Style: discordgo.PrimaryButton},
		discordgo.Button{CustomID: "fugir:" + userID, Label: "🏃 Fugir", Style: discordgo.SecondaryButton},
	}}})
}
func handleLegendaries(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p, ok := getPlayer(iUser(i).ID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}

	status := func(required int) string {
		if len(p.Badges) >= required {
			return "🟢 Disponível para aparecer"
		}
		return fmt.Sprintf("🔒 Requer %d insígnias", required)
	}

	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "🌟 Caçada Lendária",
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: officialArtworkURL(150)},
		Description: "Pokémon lendários não aparecem no encontro comum. Quando você cumprir os requisitos, eles podem surgir aleatoriamente durante /procurar.",
		Color: 0x9B59B6,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "❄️ Articuno", Value: fmt.Sprintf("%s\nLv. 30 • Chance de encontro: 0,5%% • Captura: 8%%", status(3))},
			{Name: "⚡ Zapdos", Value: fmt.Sprintf("%s\nLv. 30 • Chance de encontro: 0,5%% • Captura: 8%%", status(4))},
			{Name: "🔥 Moltres", Value: fmt.Sprintf("%s\nLv. 30 • Chance de encontro: 0,5%% • Captura: 8%%", status(7))},
			{Name: "🧬 Mewtwo", Value: fmt.Sprintf("%s\nLv. 50 • Chance de encontro: 0,2%% • Captura: 8%%", status(8))},
			{Name: "✨ Mew", Value: fmt.Sprintf("%s\nLv. 40 • Chance de encontro: 0,2%% • Captura: 8%%", status(8))},
		},
	})
}

func handleCatch(s *discordgo.Session, i *discordgo.InteractionCreate) {
	pk, ok := catchPokemon(iUser(i).ID)
	if ok {
		respond(s, i, fmt.Sprintf("🎉 **%s capturado!** Você ganhou **+10 XP** e **+5 🪙 Coins**. Use /pokemon para vê-lo.", pk.Name))
		return
	}
	respond(s, i, "❌ Não há um Pokémon válido para capturar agora, ou a captura falhou. Use /procurar para tentar novamente.")
}
func handleFlee(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if flee(iUser(i).ID) {
		respond(s, i, "🏃 Você fugiu do encontro. Use `/procurar` para procurar outro Pokémon.")
		return
	}
	respond(s, i, "Não há encontro ativo.")
}

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	userID := iUser(i).ID

	if len(parts) == 3 && parts[2] == userID && parts[0] == "rota" {
		handleRouteSelection(s, i, parts[1])
		return
	}
	if len(parts) == 3 && parts[2] == userID && parts[0] == "elite" && parts[1] == "start" {
		startEliteBattle(s,i)
		return
	}

	if len(parts) != 2 || parts[1] != userID {
		respond(s, i, "Esse botão pertence a outro treinador.")
		return
	}

	switch parts[0] {
	case "capturar":
		deferComponent(s, i)
		pk, ok := catchPokemon(userID)
		if ok {
			editComponent(s, i, fmt.Sprintf("🎉 **%s capturado!** Você ganhou **+10 XP** e **+5 🪙 Coins**.", pk.Name))
		} else {
			editComponent(s, i, "❌ **A captura falhou!** O encontro terminou e o Pokémon fugiu.")
		}
	case "fugir":
		deferComponent(s, i)
		if flee(userID) {
			editComponent(s, i, "🏃 **Você fugiu do encontro.** Use /procurar para tentar novamente.")
		} else {
			editComponent(s, i, "O encontro já terminou.")
		}
	case "batalhar":
		deferComponent(s, i)
		handleBattleAttack(s, i, userID, 0)
	case "golpe0":
		deferComponent(s, i)
		handleBattleAttack(s, i, userID, 0)
	case "golpe1":
		deferComponent(s, i)
		handleBattleAttack(s, i, userID, 1)
	case "golpe2":
		deferComponent(s, i)
		handleBattleAttack(s, i, userID, 2)
	}
}
func catchPokemon(userID string) (Pokemon, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	e, ok := store.Encounters[userID]
	if !ok || time.Now().After(e.ExpiresAt) {
		delete(store.Encounters, userID)
		return Pokemon{}, false
	}
	captureChance := captureChanceFor(e.Pokemon)
	p := store.Players[userID]
	if !p.ActiveLureUntil.IsZero() && time.Now().Before(p.ActiveLureUntil) {
		switch p.ActiveLureName {
		case "Isca":
			captureChance += 10
		case "Super Isca":
			captureChance += 18
		}
	}
	if captureChance > 90 {
		captureChance = 90
	}
	if rand.Intn(100) >= captureChance {
		delete(store.Encounters, userID)
		return Pokemon{}, false
	}
	caught := e.Pokemon
	caught.XP = 0
	caught.XPToNext = xpToNext(caught.Level)
	p.Pokemon = append(p.Pokemon, caught)
	p.Coins += 5
	p.XP += 10
	for p.XP >= p.Level*50 {
		p.XP -= p.Level * 50
		p.Level++
	}
	store.Players[userID] = p
	delete(store.Encounters, userID)
	_ = saveStoreLocked()
	return e.Pokemon, true
}

func flee(userID string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.Encounters[userID]; !ok {
		return false
	}
	delete(store.Encounters, userID)
	return true
}

func trainPokemon(userID string, index int) (string, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	p, ok := store.Players[userID]
	if !ok {
		return "Você ainda não começou. Use /iniciar primeiro.", false
	}
	if index < 0 || index >= len(p.Pokemon) {
		return fmt.Sprintf("❌ Pokémon inválido. Use /pokemon para ver os números de 1 a %d.", len(p.Pokemon)), false
	}

	pk := &p.Pokemon[index]
	pk.XP += 10
	if pk.XPToNext <= 0 {
		pk.XPToNext = xpToNext(pk.Level)
	}

	messages := []string{fmt.Sprintf("💪 **%s ganhou +10 XP!**", pk.Name)}
	for pk.XP >= pk.XPToNext {
		pk.XP -= pk.XPToNext
		pk.Level++
		pk.XPToNext = xpToNext(pk.Level)
		messages = append(messages, fmt.Sprintf("⬆️ **%s chegou ao Level %d!**", pk.Name, pk.Level))

		if evo := evolutionFor(*pk); evo != nil && pk.Level >= evo.Level {
			oldName := pk.Name
			pk.ID = evo.ToID
			pk.Name = evo.ToName
			pk.XPToNext = xpToNext(pk.Level)
			messages = append(messages, fmt.Sprintf("✨ **%s evoluiu para %s!**", oldName, pk.Name))
		}
	}

	store.Players[userID] = p
	if err := saveStoreLocked(); err != nil {
		log.Printf("save training: %v", err)
	}
	return strings.Join(messages, "\n"), true
}

func xpToNext(level int) int {
	return 10 + level*4
}

func evolutionFor(pk Pokemon) *Evolution {
	evolutions := []Evolution{
		{FromID: 1, FromName: "Bulbasaur", ToID: 2, ToName: "Ivysaur", Level: 16},
		{FromID: 2, FromName: "Ivysaur", ToID: 3, ToName: "Venusaur", Level: 32},
		{FromID: 4, FromName: "Charmander", ToID: 5, ToName: "Charmeleon", Level: 16},
		{FromID: 5, FromName: "Charmeleon", ToID: 6, ToName: "Charizard", Level: 36},
		{FromID: 7, FromName: "Squirtle", ToID: 8, ToName: "Wartortle", Level: 16},
		{FromID: 8, FromName: "Wartortle", ToID: 9, ToName: "Blastoise", Level: 36},
		{FromID: 10, FromName: "Caterpie", ToID: 11, ToName: "Metapod", Level: 7},
		{FromID: 11, FromName: "Metapod", ToID: 12, ToName: "Butterfree", Level: 10},
		{FromID: 13, FromName: "Weedle", ToID: 14, ToName: "Kakuna", Level: 7},
		{FromID: 14, FromName: "Kakuna", ToID: 15, ToName: "Beedrill", Level: 10},
		{FromID: 16, FromName: "Pidgey", ToID: 17, ToName: "Pidgeotto", Level: 18},
		{FromID: 17, FromName: "Pidgeotto", ToID: 18, ToName: "Pidgeot", Level: 36},
		{FromID: 19, FromName: "Rattata", ToID: 20, ToName: "Raticate", Level: 20},
		{FromID: 21, FromName: "Spearow", ToID: 22, ToName: "Fearow", Level: 20},
		{FromID: 41, FromName: "Zubat", ToID: 42, ToName: "Golbat", Level: 22},
		{FromID: 43, FromName: "Oddish", ToID: 44, ToName: "Gloom", Level: 21},
		{FromID: 44, FromName: "Gloom", ToID: 45, ToName: "Vileplume", Level: 32},
		{FromID: 52, FromName: "Meowth", ToID: 53, ToName: "Persian", Level: 28},
		{FromID: 29, FromName: "Nidoran♀", ToID: 30, ToName: "Nidorina", Level: 16},
		{FromID: 32, FromName: "Nidoran♂", ToID: 33, ToName: "Nidorino", Level: 16},
		{FromID: 74, FromName: "Geodude", ToID: 75, ToName: "Graveler", Level: 25},
	}
	for _, evo := range evolutions {
		if pk.ID == evo.FromID {
			copy := evo
			return &copy
		}
	}
	return nil
}

func handleRoutes(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := iUser(i).ID
	player, ok := getPlayer(userID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use /iniciar primeiro.")
		return
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(routes))
	buttons := make([]discordgo.MessageComponent, 0, 5)
	for _, route := range routes {
		if player.Level >= route.UnlockLevel {
			status := "✅ Desbloqueada"
			if player.CurrentRoute == route.ID { status = "📍 **Atual**" }
			fields = append(fields, &discordgo.MessageEmbedField{Name: route.Name, Value: fmt.Sprintf("%s\nNíveis: %d–%d\n%s", status, route.MinLevel, route.MaxLevel, route.Description)})
			buttons = append(buttons, discordgo.Button{CustomID: "rota:" + route.ID + ":" + userID, Label: route.Name, Style: discordgo.PrimaryButton})
		} else {
			fields = append(fields, &discordgo.MessageEmbedField{Name: route.Name, Value: fmt.Sprintf("🔒 Desbloqueia no Level %d\nNíveis: %d–%d", route.UnlockLevel, route.MinLevel, route.MaxLevel)})
		}
	}
	respondEmbedWithComponents(s, i, &discordgo.MessageEmbed{Title: "🗺️ Rotas", Description: fmt.Sprintf("Você está em **%s**. Escolha onde quer procurar Pokémon.", routeName(player.CurrentRoute)), Color: 0x57F287, Fields: fields}, []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}})
}

func handleRouteSelection(s *discordgo.Session, i *discordgo.InteractionCreate, routeID string) {
	deferComponent(s, i)
	userID := iUser(i).ID
	player, ok := getPlayer(userID)
	if !ok { editComponent(s, i, "Você ainda não começou. Use /iniciar primeiro."); return }
	route := getRoute(routeID)
	if route == nil { editComponent(s, i, "Essa rota não existe."); return }
	if player.Level < route.UnlockLevel { editComponent(s, i, fmt.Sprintf("🔒 **%s** desbloqueia no Level %d.", route.Name, route.UnlockLevel)); return }
	store.mu.Lock()
	player.CurrentRoute = route.ID
	store.Players[userID] = player
	store.mu.Unlock()
	if err := saveStore(); err != nil { log.Printf("save route selection: %v", err) }
	editComponent(s, i, fmt.Sprintf("📍 **Rota alterada!** Agora você está em **%s**. Use /procurar para encontrar Pokémon nessa rota.", route.Name))
}

func getRoute(id string) *Route {
	for idx := range routes { if routes[idx].ID == id { return &routes[idx] } }
	return nil
}

func routeName(id string) string {
	if route := getRoute(id); route != nil { return route.Name }
	return "Route 1"
}

func randomPokemonForRoute(route Route, owned []Pokemon) Pokemon {
	// Prioriza espécies ainda não capturadas, mas respeita a raridade:
	// comuns aparecem bastante; raros e muito raros aparecem bem menos.
	available := make([]int, 0, len(route.PokemonIDs))
	for _, id := range route.PokemonIDs {
		if !hasPokemonID(owned, id) {
			available = append(available, id)
		}
	}
	if len(available) == 0 {
		available = route.PokemonIDs
	}

	pokemonID := weightedPokemonID(available)
	pk := pokemonByID(pokemonID)
	pk.Level = route.MinLevel + rand.Intn(route.MaxLevel-route.MinLevel+1)
	pk.XPToNext = 10 + pk.Level*4
	pk.Shiny = rand.Intn(100) == 0
	return pk
}

func weightedPokemonID(ids []int) int {
	total := 0
	for _, id := range ids {
		total += encounterWeight(id)
	}
	if total <= 0 {
		return ids[rand.Intn(len(ids))]
	}

	roll := rand.Intn(total)
	for _, id := range ids {
		weight := encounterWeight(id)
		if roll < weight {
			return id
		}
		roll -= weight
	}
	return ids[len(ids)-1]
}

func encounterWeight(id int) int {
	switch rarityByID(id) {
	case "Comum":
		return 70
	case "Incomum":
		return 22
	case "Raro":
		return 7
	case "Muito raro":
		return 1
	default:
		return 70
	}
}

func hasPokemonID(pokemon []Pokemon, id int) bool {
	for _, pk := range pokemon {
		if pk.ID == id {
			return true
		}
	}
	return false
}

func pokemonByID(id int) Pokemon {
	for _, pk := range pokemonPool {
		if pk.ID == id {
			pk.Type = pokemonType(pk.ID)
			return pk
		}
	}
	pk := pokemonPool[0]
	pk.Type = pokemonType(pk.ID)
	return pk
}

func deferComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
}

func editComponent(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	emptyComponents := []discordgo.MessageComponent{}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Components: &emptyComponents})
	if err != nil { log.Printf("edit component response: %v", err) }
}
func newPokemonByName(name string) Pokemon {
	for _, pk := range pokemonPool {
		if strings.EqualFold(pk.Name, name) {
			pk.Type = pokemonType(pk.ID)
			return pk
		}
	}
	pk := pokemonPool[1]
	pk.Type = pokemonType(pk.ID)
	return pk
}

func spriteURL(id int) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png", id)
}

func officialArtworkURL(id int) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/official-artwork/%d.png", id)
}

func rarity(pk Pokemon) string {
	if pk.Shiny {
		return "✨ Shiny — extremamente raro"
	}
	return rarityByID(pk.ID)
}

func rarityByID(id int) string {
	if isLegendary(id) {
		return "Lendário"
	}
	switch id {
	case 128, 123:
		return "Muito raro"
	case 1, 4, 7, 102:
		return "Raro"
	case 25, 35, 52:
		return "Incomum"
	default:
		return "Comum"
	}
}

func isLegendary(id int) bool {
	switch id {
	case 144, 145, 146, 150, 151:
		return true
	default:
		return false
	}
}

func legendaryEncounter(player Player) (Pokemon, bool) {
	badges := len(player.Badges)
	chance := 0
	ids := []int{}

	switch {
	case badges >= 8:
		if rand.Intn(1000) < 2 {
			ids = []int{150}
			chance = 1
		} else if rand.Intn(1000) < 2 {
			ids = []int{151}
			chance = 1
		}
	case badges >= 7:
		if rand.Intn(1000) < 5 {
			ids = []int{146}
			chance = 1
		}
	case badges >= 4:
		if rand.Intn(1000) < 5 {
			ids = []int{145}
			chance = 1
		}
	case badges >= 3:
		if rand.Intn(1000) < 5 {
			ids = []int{144}
			chance = 1
		}
	}

	if chance == 0 || len(ids) == 0 {
		return Pokemon{}, false
	}

	pk := pokemonByID(ids[rand.Intn(len(ids))])
	if pk.ID == 150 {
		pk.Level = 50
	} else if pk.ID == 151 {
		pk.Level = 40
	} else {
		pk.Level = 30
	}
	pk.XPToNext = xpToNext(pk.Level)
	pk.Shiny = false
	pk.Type = pokemonType(pk.ID)
	return pk, true
}

func captureChanceFor(pk Pokemon) int {
	var chance int
	switch rarityByID(pk.ID) {
	case "Comum":
		chance = 75
	case "Incomum":
		chance = 60
	case "Raro":
		chance = 42
	case "Muito raro":
		chance = 25
	case "Lendário":
		chance = 8
	default:
		chance = 75
	}

	if pk.Shiny {
		chance = 15
	}

	// Pokémon de níveis mais altos são um pouco mais difíceis de capturar.
	chance -= (pk.Level - 1) / 4
	if chance < 5 {
		chance = 5
	}
	return chance
}

func iUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

func getPlayer(id string) (Player, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	p, ok := store.Players[id]
	return p, ok
}

func loadStore() error {
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		return os.MkdirAll(filepath.Dir(dataFile), 0755)
	}
	b, err := os.ReadFile(dataFile)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := json.Unmarshal(b, &store.Players); err != nil { return err }
	for id, p := range store.Players {
		if len(p.Team) == 0 && len(p.Pokemon) > 0 { p.Team = []int{0}; store.Players[id] = p }
	}
	return nil
}

func saveStore() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return saveStoreLocked()
}

func saveStoreLocked() error {
	if err := os.MkdirAll(filepath.Dir(dataFile), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(store.Players, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFile, b, 0644)
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content},
	})
}


func progressBar(current, max, width int) string {
	if max <= 0 { max = 1 }
	if current < 0 { current = 0 }
	if current > max { current = max }
	if width <= 0 { width = 10 }
	filled := int(float64(current) / float64(max) * float64(width))
	return strings.Repeat("🟩", filled) + strings.Repeat("⬛", width-filled)
}

func typeEmoji(t string) string {
	switch {
	case strings.Contains(t, "Fogo"): return "🔥"
	case strings.Contains(t, "Água"): return "💧"
	case strings.Contains(t, "Elétrico"): return "⚡"
	case strings.Contains(t, "Planta"): return "🌿"
	case strings.Contains(t, "Pedra"): return "🪨"
	case strings.Contains(t, "Terra"): return "🌍"
	case strings.Contains(t, "Veneno"): return "☠️"
	case strings.Contains(t, "Psíquico"): return "🔮"
	case strings.Contains(t, "Gelo"): return "❄️"
	case strings.Contains(t, "Fantasma"): return "👻"
	case strings.Contains(t, "Lutador"): return "🥊"
	case strings.Contains(t, "Dragão"): return "🐉"
	case strings.Contains(t, "Inseto"): return "🐛"
	case strings.Contains(t, "Voador"): return "🪽"
	default: return "🔹"
	}
}

func typeColor(t string) int {
	switch {
	case strings.Contains(t, "Fogo"): return 0xE74C3C
	case strings.Contains(t, "Água"): return 0x3498DB
	case strings.Contains(t, "Elétrico"): return 0xF1C40F
	case strings.Contains(t, "Planta"): return 0x2ECC71
	case strings.Contains(t, "Pedra"): return 0x8E7D5B
	case strings.Contains(t, "Terra"): return 0xA67C52
	case strings.Contains(t, "Veneno"): return 0x9B59B6
	case strings.Contains(t, "Psíquico"): return 0xE84393
	case strings.Contains(t, "Gelo"): return 0x74B9FF
	case strings.Contains(t, "Fantasma"): return 0x6C5CE7
	case strings.Contains(t, "Lutador"): return 0xD35400
	case strings.Contains(t, "Dragão"): return 0x5F27CD
	default: return 0x5865F2
	}
}

func gymAce(g Gym) Pokemon {
	if len(g.Pokemon) == 0 { return Pokemon{ID:25, Name:"Pikachu", Type:"Elétrico", Level:1} }
	pk := g.Pokemon[len(g.Pokemon)-1]
	pk.Type = pokemonType(pk.ID)
	return pk
}

func leagueStatus(p Player, idx int) string {
	if p.Champion || idx < p.LeagueWins { return "✅ Concluído" }
	if len(p.Badges) < 8 { return "🔒 8 insígnias" }
	if idx == p.LeagueWins { return "⚔️ Próximo desafio" }
	return "🔒 Aguarde a etapa anterior"
}

func trainerSpriteURL(name string) string {
	slugs := map[string]string{
		"Brock":"brock", "Misty":"misty", "Lt. Surge":"lt-surge", "Erika":"erika",
		"Koga":"koga", "Sabrina":"sabrina", "Blaine":"blaine", "Giovanni":"giovanni",
		"Lorelei":"lorelei", "Bruno":"bruno", "Agatha":"agatha", "Lance":"lance", "Campeão Blue":"blue",
	}
	if slug, ok := slugs[name]; ok {
		return "https://img.pokemondb.net/sprites/trainers/heartgold-soulsilver/" + slug + ".png"
	}
	return "https://img.pokemondb.net/sprites/trainers/heartgold-soulsilver/red.png"
}

func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	respondEmbedWithComponents(s, i, embed, nil)
}

func respondEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}