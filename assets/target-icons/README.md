# Investment target icons

The 13 preset targets and the custom-target fallback have distinct original icons generated with the built-in image tool. Its underlying model cannot be selected or verified as Image 2.5.

- `source/`: original transparent PNGs, retained for future regeneration and resizing.
- `../../web/public/target-icons/`: 128 x 128 PNGs served by the app (about 224 KiB total).
- `prompts.json`: the full prompt set and shared visual guidance.
- `generation.json`: the actual generation/edit prompts and source provenance for the selected outputs.

QDII index icons use a globe. The corresponding US index icons retain the same subject with a coral star in place of the globe. The front end supplies the rounded background using the selected theme; no background tile is baked into the PNG.

To rebuild the optimized assets on Windows, run `./assets/target-icons/prepare-assets.ps1` from the repository root. Resizing preserves the alpha channel and does not alter the original files.
