var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
(() => __awaiter(this, void 0, void 0, function* () {
    function loadData(path, knownDataSize, onProgress) {
        return __awaiter(this, void 0, void 0, function* () {
            const response = yield fetch(path);
            let dataSize = parseInt(response.headers.get('Content-Length'));
            let contentLengthMissing = false;
            if (isNaN(dataSize)) {
                dataSize = knownDataSize;
                contentLengthMissing = true;
            }
            let data = [];
            let dataRead = 0;
            const reader = response.body.getReader();
            while (true) {
                const bodyRead = yield reader.read();
                if (bodyRead.done) {
                    break;
                }
                else {
                    dataRead += bodyRead.value.length;
                    for (let i = 0; i < bodyRead.value.length; i++) {
                        data.push(bodyRead.value[i]);
                    }
                    onProgress(dataRead, dataSize, contentLengthMissing);
                }
            }
            return new Uint8Array(data);
        });
    }
    const loadingScreen = document.getElementById('loading-screen');
    const progressBar = document.getElementById('progress-bar');
    const progressFill = document.getElementById('progress-fill');
    const errorTextBox = document.getElementById('error-text-box');
    const errorText = document.getElementById('error-text');
    try {
        const data = yield loadData('minesweeper.wasm', MINESWEEPER_WASM_SIZE, (readData, dataSize, contentLengthMissing) => {
            const progress = readData / dataSize;
            progressFill.style.width = `${progress * 100}%`;
        });
        document.body.removeChild(loadingScreen);
        const go = new Go();
        WebAssembly.instantiate(new Uint8Array(data), go.importObject).then(instnace => {
            go.run(instnace.instance);
        });
    }
    catch (err) {
        errorTextBox.style.display = 'block'; // show errorTextBox
        errorText.innerText = err.toString();
        console.error(err);
    }
}))();
