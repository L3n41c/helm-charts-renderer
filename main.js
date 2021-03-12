(async function loadAndRunGoWasm() {
    const go = new Go();

    const buffer = pako.ungzip(await (await fetch("main.wasm.gz")).arrayBuffer());

    // A fetched response might be decompressed twice on Firefox.
    // See https://bugzilla.mozilla.org/show_bug.cgi?id=610679
    if (buffer[0] === 0x1f && buffer[1] === 0x8b) {
        buffer = pako.ungzip(buffer);
    }

    const result = await WebAssembly.instantiate(buffer, go.importObject);
    go.run(result.instance);
})()
