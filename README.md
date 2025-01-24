# Minesweeper

![screenshot](readme_stuff/screenshot.png)

Minesweeper written in go.

# How to build

### Dependencies

On **Windows**, you only need go compiler.

On **Linux**, you'll need dependencies for [Ebitengine](https://ebitengine.org/en/documents/install.html?os=linux).

You can learn how to compile Ebitengine applications [here](https://ebitengine.org/en/documents/install.html?os=linux).

### Building Desktop Version
```
go run build.go
```

### Building Web Version
```
go run build.go web
```

I have included simple a static server.

Do
```
go run run_web.go
```

But if you don't like mine, you can just serve web_build directory with whatever static server you like.

# Credits

### Used sound effects

[Interface Sounds Starter Pack](https://opengameart.org/content/interface-sounds-starter-pack) by p0ss - License : CC BY-SA 3.0

[Pop!.wav](https://freesound.org/people/kwahmah_02/sounds/260614/) by kwahmah_02 - License : CC BY 3.0

[Fabric flaps](https://freesound.org/people/PelicanPolice/sounds/580967/) by PelicanPolice - License : CC0 1.0

[Cloth Flaps](https://freesound.org/people/Sauron974/sounds/188733/) by Sauron974 - License : CC BY 4.0

[Swish - bamboo stick weapon swhoshes](https://opengameart.org/content/swish-bamboo-stick-weapon-swhoshes) by qubodup - License : CC0 1.0

[51 UI sound effects (buttons, switches and clicks)](https://opengameart.org/content/51-ui-sound-effects-buttons-switches-and-clicks) by Kenney - License : CC0 1.0

# License

This project is under MIT License.

Sound effects in assets_sound directory are under CC BY-SA 4.0.
