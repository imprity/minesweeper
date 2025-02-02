declare var Go : any
declare var MINESWEEPER_WASM_SIZE : number

(async ()=>{
    async function loadData(
        path : string,
        knownDataSize : number,
        onProgress : (readData : number, dataSize : number, contentLengthMissing : boolean) => void
    ) : Promise<Uint8Array>{
        const response = await fetch(path)

        let dataSize : number = parseInt(response.headers.get('Content-Length'))
        let contentLengthMissing : boolean = false

        if (isNaN(dataSize)) {
            dataSize = knownDataSize
            contentLengthMissing = true
        }

        let data = []
        let dataRead : number = 0

        const reader = response.body.getReader()

        while (true) {
            const bodyRead = await reader.read()

            if (bodyRead.done) {
                break
            }else {
                dataRead += bodyRead.value.length

                for (let i=0; i<bodyRead.value.length; i++) {
                    data.push(bodyRead.value[i])
                }

                onProgress(dataRead, dataSize, contentLengthMissing)
            }
        }

        return new Uint8Array(data)
    }

    // get document elements
    const loadingScreen = document.getElementById('loading-screen')

    const progressBar = document.getElementById('progress-bar')
    const progressFill = document.getElementById('progress-fill')

    const loadingErrorContainer = document.getElementById('loading-error-container')
    const loadingErrorText = document.getElementById('loading-error-text')

    const crashScreen = document.getElementById('crash-screen')
    const crashErrortext = document.getElementById('crash-error-text')

    // load wasm
    try {
        const data = await loadData(
            'minesweeper.wasm',
            MINESWEEPER_WASM_SIZE,
            (readData : number, dataSize : number, contentLengthMissing : boolean) => {
                const progress = readData / dataSize
                progressFill.style.width = `${progress * 100}%`
            }
        )

        document.body.removeChild(loadingScreen)

        // before instantiating go wasm
        // override few methods and syscalls

        function countNewLine (str : string) : number {
            let counter = 0
            let index = 0
            while (true) {
                const newIndex = str.indexOf("\n", index)
                if (newIndex < 0) {
                    break
                }else {
                    counter++
                }
                index = newIndex + 1
            }

            return counter
        }

        const originalWriteSync = globalThis.fs.writeSync
	    const decoder = new TextDecoder("utf-8");

        let logText = ""
        const logTextMax = 500

        let newLineCounter = 0

        globalThis.fs.writeSync = function(fd, buf) {
            let bufText = decoder.decode(buf)
            const newLineCount = countNewLine(bufText)
            newLineCounter += newLineCount

            logText += bufText

            if (newLineCounter > logTextMax) {
                const diff = newLineCounter - logTextMax
                let index = 0
                for (let i=0; i<diff; i++) {
                    const newIndex = logText.indexOf("\n", index)
                    if (newIndex < 0) {
                        break
                    }
                    index = newIndex + 1
                }
                if (index < 0) {
                    index = 0
                }

                logText = logText.substring(index)

                newLineCounter = logTextMax
            }

            return originalWriteSync(fd, buf)
        }

        const go = new Go()
        go.exit = (code) => {
            if (code != 0) {
                crashScreen.style.display = 'flex' // show crashScreen
                crashErrortext.innerText = logText
            }
        }

        WebAssembly.instantiate(new Uint8Array(data), go.importObject).then(instnace=>{
            go.run(instnace.instance)
        })
    } catch(err) {
        loadingErrorContainer.style.display = 'block' // show error container
        loadingErrorText.innerText = err.toString()
        console.error(err)
    }
})()
