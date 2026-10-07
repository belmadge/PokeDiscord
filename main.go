package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
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
	Level    int    `json:"level"`
	XP       int    `json:"xp"`
	XPToNext int    `json:"xp_to_next"`
	Shiny    bool   `json:"shiny"`
}

type Player struct {
	DiscordID    string    `json:"discord_id"`
	Username     string    `json:"username"`
	Coins        int       `json:"coins"`
	Level        int       `json:"level"`
	XP           int       `json:"xp"`
	Pokemon      []Pokemon `json:"pokemon"`
	CurrentRoute string    `json:"current_route"`
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

type Encounter struct {
	OwnerID   string
	Pokemon   Pokemon
	ExpiresAt time.Time
}

type Store struct {
	mu         sync.RWMutex
	Players    map[string]Player
	Encounters map[string]Encounter
}

var store = &Store{Players: map[string]Player{}, Encounters: map[string]Encounter{}}

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
		{Name: "procurar", Description: "Procure um Pokémon selvagem"},
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
	case "procurar":
		handleHunt(s, i)
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
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Próximo passo", Value: "Use `/procurar` para procurar um Pokémon selvagem."},
			{Name: "💰 Coins", Value: "100"},
		},
	})
}

func handleProfile(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p, ok := getPlayer(iUser(i).ID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use `/iniciar` primeiro.")
		return
	}
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "👤 " + p.Username, Description: "Treinador Pokémon", Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Level", Value: strconv.Itoa(p.Level), Inline: true},
			{Name: "✨ XP", Value: strconv.Itoa(p.XP), Inline: true},
			{Name: "💰 Coins", Value: strconv.Itoa(p.Coins), Inline: true},
			{Name: "📦 Pokémon", Value: strconv.Itoa(len(p.Pokemon)), Inline: true},
		},
	})
}

func handlePokemon(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p, ok := getPlayer(iUser(i).ID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use `/iniciar` primeiro.")
		return
	}
	lines := make([]string, 0, len(p.Pokemon))
	for idx, pk := range p.Pokemon {
		shiny := ""
		if pk.Shiny {
			shiny = " ✨"
		}
		lines = append(lines, fmt.Sprintf("**%d. %s**%s — Lv. %d — XP %d/%d", idx+1, pk.Name, shiny, pk.Level, pk.XP, pk.XPToNext))
	}
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "📦 Seus Pokémon", Description: strings.Join(lines, "\n"), Color: 0x57F287,
	})
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

	pk := randomPokemonForRoute(*route, player.Pokemon)
	store.Encounters[userID] = Encounter{OwnerID: userID, Pokemon: pk, ExpiresAt: time.Now().Add(60 * time.Second)}
	store.mu.Unlock()

	respondEmbedWithComponents(s, i, &discordgo.MessageEmbed{
		Title: "🌿 Pokémon selvagem apareceu!",
		Description: fmt.Sprintf("Um **%s** selvagem apareceu na **%s**!", pk.Name, route.Name),
		Color: 0xFEE75C,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: spriteURL(pk.ID)},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Level", Value: strconv.Itoa(pk.Level), Inline: true},
			{Name: "Raridade", Value: rarity(pk), Inline: true},
			{Name: "🗺️ Rota", Value: route.Name, Inline: true},
			{Name: "⏱️ Encontro", Value: "60 segundos", Inline: true},
		},
	}, []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: "capturar:" + userID, Label: "🎯 Capturar", Style: discordgo.PrimaryButton},
		discordgo.Button{CustomID: "fugir:" + userID, Label: "🏃 Fugir", Style: discordgo.SecondaryButton},
	}}})
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
	if rand.Intn(100) >= 70 {
		delete(store.Encounters, userID)
		return Pokemon{}, false
	}

	p := store.Players[userID]
	p.Pokemon = append(p.Pokemon, e.Pokemon)
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
	// Prioriza espécies que o treinador ainda não capturou nesta rota.
	// Depois que completar a lista da rota, os encontros voltam a ser aleatórios.
	available := make([]int, 0, len(route.PokemonIDs))
	for _, id := range route.PokemonIDs {
		if !hasPokemonID(owned, id) {
			available = append(available, id)
		}
	}
	if len(available) == 0 {
		available = route.PokemonIDs
	}

	pokemonID := available[rand.Intn(len(available))]
	pk := pokemonByID(pokemonID)
	pk.Level = route.MinLevel + rand.Intn(route.MaxLevel-route.MinLevel+1)
	pk.XPToNext = 10 + pk.Level*4
	pk.Shiny = rand.Intn(100) == 0
	return pk
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
	for _, pk := range pokemonPool { if pk.ID == id { return pk } }
	return pokemonPool[0]
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
			return pk
		}
	}
	return pokemonPool[1]
}

func spriteURL(id int) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png", id)
}

func rarity(pk Pokemon) string {
	if pk.Shiny {
		return "✨ Shiny!"
	}
	if pk.Name == "Pikachu" {
		return "⭐ Raro"
	}
	return "Comum"
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
	return json.Unmarshal(b, &store.Players)
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

func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	respondEmbedWithComponents(s, i, embed, nil)
}

func respondEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}
