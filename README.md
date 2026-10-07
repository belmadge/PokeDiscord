# PokeDiscord

A small Pokémon RPG bot for Discord, built in Go.

## MVP

- `/iniciar` — creates your trainer and chooses Bulbasaur, Charmander or Squirtle
- `/perfil` — trainer level, XP, coins and collection size
- `/pokemon` — lists your Pokémon
- `/procurar` — generates a wild encounter
- `/capturar` — attempts capture
- `/fugir` — ends the encounter
- Buttons for capture/flee
- Local JSON persistence in `data/players.json`
- Pokémon sprites from PokeAPI

## Run locally

1. Create a Discord application and bot in the Discord Developer Portal.
2. Invite it to your server with the scopes `bot` and `applications.commands`.
3. Give it permission to send messages and use application commands.
4. Set `DISCORD_TOKEN`.
5. Optionally set `DISCORD_GUILD_ID` for instant slash-command registration in one server.

PowerShell:

```powershell
$env:DISCORD_TOKEN="YOUR_TOKEN"
$env:DISCORD_GUILD_ID="YOUR_SERVER_ID"
go mod tidy
go run .
```

If `DISCORD_GUILD_ID` is omitted, commands are registered globally.

## First test

```
/start starter:Charmander
/profile
/hunt
```

Then click **🎯 Capturar** or use `/capturar`.

After a successful capture:

```
/pokemon
```

## Architecture for the next iterations

```
Discord
   |
Go Bot
   |
Game Use Cases
   |
Repository
   |
PostgreSQL
```

The JSON store is deliberately temporary so the first playable version can be tested immediately. The next milestone is PostgreSQL.

## Roadmap

- [ ] PostgreSQL
- [ ] Full Pokédex / encounter tables
- [ ] Pokéballs and inventory
- [ ] Evolution
- [ ] Battle / Coliseum
- [ ] Trading
- [ ] Lures
- [ ] Safari Zone
- [ ] Leaderboards
- [ ] AWS deployment
- [ ] Observability

## Prototype note

This project is initially intended for private use with friends. Pokémon names and assets are used for the prototype; review intellectual-property/licensing requirements before turning it into a public or commercial product.
