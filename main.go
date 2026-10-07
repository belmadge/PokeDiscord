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
	DiscordID string    `json:"discord_id"`
	Username  string    `json:"username"`
	Coins     int       `json:"coins"`
	Level     int       `json:"level"`
	XP        int       `json:"xp"`
	Pokemon   []Pokemon `json:"pokemon"`
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
	{ID: 19, Name: "Rattata", Level: 1, XPToNext: 14},
	{ID: 21, Name: "Spearow", Level: 1, XPToNext: 14},
	{ID: 25, Name: "Pikachu", Level: 1, XPToNext: 14},
	{ID: 43, Name: "Oddish", Level: 1, XPToNext: 14},
	{ID: 52, Name: "Meowth", Level: 1, XPToNext: 14},
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
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
			Name: "start", Description: "Comece sua jornada Pokémon",
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
		{Name: "profile", Description: "Veja seu perfil de treinador"},
		{Name: "pokemon", Description: "Veja seus Pokémon"},
		{Name: "hunt", Description: "Procure um Pokémon selvagem"},
		{Name: "catch", Description: "Tente capturar o Pokémon encontrado"},
		{Name: "flee", Description: "Fuja do encontro atual"},
	}
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	return err
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
	case "start":
		handleStart(s, i)
	case "profile":
		handleProfile(s, i)
	case "pokemon":
		handlePokemon(s, i)
	case "hunt":
		handleHunt(s, i)
	case "catch":
		handleCatch(s, i)
	case "flee":
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
		respond(s, i, "Você já começou sua jornada! Use `/profile`.")
		return
	}
	p := Player{
		DiscordID: user.ID, Username: user.Username, Coins: 100, Level: 1,
		Pokemon: []Pokemon{newPokemonByName(starter)},
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
			{Name: "Próximo passo", Value: "Use `/hunt` para procurar um Pokémon selvagem."},
			{Name: "💰 Coins", Value: "100"},
		},
	})
}

func handleProfile(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p, ok := getPlayer(iUser(i).ID)
	if !ok {
		respond(s, i, "Você ainda não começou. Use `/start` primeiro.")
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
		respond(s, i, "Você ainda não começou. Use `/start` primeiro.")
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
	if _, ok := store.Players[userID]; !ok {
		store.mu.Unlock()
		respond(s, i, "Você ainda não começou. Use `/start` primeiro.")
		return
	}
	if e, ok := store.Encounters[userID]; ok && time.Now().Before(e.ExpiresAt) {
		store.mu.Unlock()
		respond(s, i, "Você já está em um encontro! Use `/catch` ou `/flee`.")
		return
	}

	pk := pokemonPool[rng.Intn(len(pokemonPool))]
	pk.Level = 1 + rng.Intn(5)
	pk.XPToNext = 10 + pk.Level*4
	pk.Shiny = rng.Intn(100) == 0
	store.Encounters[userID] = Encounter{OwnerID: userID, Pokemon: pk, ExpiresAt: time.Now().Add(60 * time.Second)}
	store.mu.Unlock()

	respondEmbedWithComponents(s, i, &discordgo.MessageEmbed{
		Title: "🌿 Pokémon selvagem apareceu!",
		Description: fmt.Sprintf("Um **%s** selvagem apareceu na **Route 1**!", pk.Name),
		Color: 0xFEE75C,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: spriteURL(pk.ID)},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Level", Value: strconv.Itoa(pk.Level), Inline: true},
			{Name: "Raridade", Value: rarity(pk), Inline: true},
			{Name: "⏱️ Encontro", Value: "60 segundos", Inline: true},
		},
	}, []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: "catch:" + userID, Label: "🎯 Capturar", Style: discordgo.PrimaryButton},
		discordgo.Button{CustomID: "flee:" + userID, Label: "🏃 Fugir", Style: discordgo.SecondaryButton},
	}}})
}

func handleCatch(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if catchPokemon(iUser(i).ID) {
		respond(s, i, "🎉 **Capturado!** O Pokémon entrou para sua coleção. Use `/pokemon` para vê-lo.")
		return
	}
	respond(s, i, "❌ Não há um Pokémon válido para capturar agora, ou a captura falhou. Use `/hunt` para tentar novamente.")
}

func handleFlee(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if flee(iUser(i).ID) {
		respond(s, i, "🏃 Você fugiu do encontro. Use `/hunt` para procurar outro Pokémon.")
		return
	}
	respond(s, i, "Não há encontro ativo.")
}

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 2 || parts[1] != iUser(i).ID {
		respond(s, i, "Esse encontro pertence a outro treinador.")
		return
	}
	switch parts[0] {
	case "catch":
		if catchPokemon(iUser(i).ID) {
			respond(s, i, "🎉 **Capturado!** O Pokémon entrou para sua coleção.")
		} else {
			respond(s, i, "❌ O encontro expirou, já foi resolvido ou a captura falhou.")
		}
	case "flee":
		if flee(iUser(i).ID) {
			respond(s, i, "🏃 Você fugiu do encontro.")
		} else {
			respond(s, i, "O encontro já terminou.")
		}
	}
}

func catchPokemon(userID string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()

	e, ok := store.Encounters[userID]
	if !ok || time.Now().After(e.ExpiresAt) {
		delete(store.Encounters, userID)
		return false
	}
	if rng.Intn(100) >= 70 {
		delete(store.Encounters, userID)
		return false
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
	return true
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
	store.mu.RLock()
	defer store.mu.RUnlock()
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
