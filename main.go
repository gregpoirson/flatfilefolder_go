package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var reader *bufio.Reader

var FILTER_EXT = "|.dll|.exe|.zip|.pdb|.db|.ico|.png|.jpg|.jpeg|.cache|"

var FILTER_PATH = [3]string{`\bin\`, `\obj\`, `\.git\`}

const C_PATH = "$@$@PATH$@$@"
const C_FILE = "$@$@FILE$@$@"

func main() {
	//testes

	//testes

	fmt.Println("L(er) ou E(screver) ou S(air) ?")

	reader = bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()

	if err != nil {
		fmt.Println(err)
	}

	switch strings.ToUpper(string(char)) {
	case "L":
		fmt.Println("L(er)")
	case "E":
		Escrever()
	default:
		fmt.Println("opção não reconhecida")
	}

	fmt.Printf("ENTER para sair...")
	reader = bufio.NewReader(os.Stdin)
	reader.ReadRune()
}

func Escrever() error {
	fmt.Println("Informa o diretorio (+ENTER):")
	reader = bufio.NewReader(os.Stdin)
	bytes, _, err := reader.ReadLine()
	if err != nil {
		fmt.Printf("err=%v", err)
		return err
	}

	path := string(bytes)

	if !dirExists(path) {
		panic(fmt.Sprintf("O Caminho não foi encontrado : <%s>\n", path))
	}

	fmt.Printf("Quer zipar o arquivo depois da geração ? S(im) / N(ão) ?")
	reader = bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		fmt.Println(err)
	}
	compressAfter := strings.ToUpper(string(char)) == "S"

	start := time.Now()
	start1 := time.Now()

	destFile := EscreverFile(path)

	elapsed1 := time.Since(start1)

	fmt.Printf("Tempo execução Escrever= %s\n", elapsed1)

	if compressAfter {
		//zipar o arquivo
		start1 = time.Now()

		CompressFile(destFile)

		fmt.Printf("Tempo execução Compress= %s\n", time.Since(start1))
	}

	elapsed := time.Since(start)
	fmt.Printf("Tempo execução Total= %s\n", elapsed)

	return nil
}

func EscreverFile(path string) string {

	allFiles := make([]string, 0)
	allFiles = getAllFilesInDir(path, allFiles)

	fmt.Printf("Processando %v arquivos...\n", len(allFiles))

	fmt.Println("Filtrando os arquivos indesejados...")
	allFiles, qtd := filterFiles(allFiles)
	fmt.Printf("Depois do filtro : %v arquivos\n", qtd)

	//ordenando os arquivos por nome
	//serve apenas para poder comparar o outro arquivo gerado pela rotina .Net
	fmt.Println("Ordenando os arquivos por nome...")
	//sort.Strings(allFiles)	//esse metodo não case insensitive!
	sort.Slice(allFiles, func(i, j int) bool { return strings.ToLower(allFiles[i]) < strings.ToLower(allFiles[j]) })

	fmt.Println("Processamento dos arquivos elegíveis...")

	//destFile := `C:\temp\test.txt`
	destFile := fmt.Sprint(path, "_go.txt")

	_, err := os.Stat(destFile)
	if !os.IsNotExist(err) {
		//remove o arquivo de destino caso exista
		err := os.Remove(destFile)
		if err != nil {
			log.Fatal("Err Remove=", err)
		}
	}

	// If the file doesn't exist, create it, or append to the file
	fDest, err := os.OpenFile(destFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Err OpenFile=", err)
	}

	//grava path original
	if _, err := fDest.Write(cryptBytes([]byte(fmt.Sprintf("%s=%s", C_PATH, path)))); err != nil {
		fDest.Close() // ignore error; Write error takes precedence
		log.Fatal("Err Write 1=", err)
	}

	qtd = 0
	for _, f := range allFiles {

		if f == "" {
			continue
		}

		qtd++

		//grava caminho arquivo atual
		//if _, err := fDest.Write([]byte(fmt.Sprintf("\r\n%s=%s\r\n", C_FILE, f))); err != nil {
		if _, err := fDest.Write(cryptBytes([]byte(fmt.Sprintf("\r\n%s=%s\r\n", C_FILE, f)))); err != nil {
			fDest.Close() // ignore error; Write error takes precedence
			log.Fatal("Err Write 2=", err)
		}

		//ler conteudo arquivo atual
		byt, err := os.ReadFile(f)
		if err != nil {
			log.Fatal("Err ReadFile=", err)
		}

		byt = cryptBytes(byt)

		if err != nil {
			fDest.Close()
			log.Fatal("Err cryptByt=", err)
		}

		//grava conteudo arquivo atual
		if _, err := fDest.Write(byt); err != nil {
			fDest.Close()
			log.Fatal("Err Write 3=", err)
		}
	}

	if err := fDest.Close(); err != nil {
		log.Fatal("Err Close file=", err)
	}

	fmt.Printf("%v arquivos processados com sucesso\n", qtd)
	fmt.Printf("Conteudo gravado no arquivo %s\n", destFile)

	return destFile
}

func cryptBytes(byt []byte) []byte {
	for i, b := range byt {
		if b == 0x00 {
			b = 0xff
		} else {
			b = b - 0x1
		}
		byt[i] = b
	}
	return byt
}

func dirExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func filterFiles(allFiles []string) ([]string, int) {
	var fp string
	var interromp = false

	for i, file := range allFiles {
		interromp = false

		fp = filepath.Ext(file)

		if strings.Contains(FILTER_EXT, fmt.Sprintf("|%s|", fp)) {
			allFiles[i] = "" //vamos marcar como vazio para não pegar
			continue
		}

		for _, vp := range FILTER_PATH {
			if strings.Contains(file, vp) {
				allFiles[i] = "" //vamos marcar como vazio para não pegar
				interromp = true
			}
		}
		if interromp {
			continue
		}

		if !checkFileIsText(file) {
			allFiles[i] = "" //vamos marcar como vazio para não pegar
			//fmt.Printf("Arquivo %s foi detectato BINARY\n", file)
			continue
		}
	}
	// for _, file := range allFiles {
	// 	if file != "" {
	// 		fmt.Println(file)
	// 	}
	// }
	qtd := 0
	for _, file := range allFiles {
		if file != "" {
			qtd++
		}
	}

	return allFiles, qtd
}

func checkFileIsText(file string) bool {
	//abre o arquivo e lê os 100 primeiros bytes
	//se for tudo ASCII, então retorna TRUE
	isText := true

	f, err := os.Open(file)
	check(file, err, 1)
	defer f.Close()

	b1 := make([]byte, 100)
	n1, err := f.Read(b1)
	check(file, err, 2)
	for i := 0; i < n1 && isText; i++ {
		isText = isText && !(IsControl(b1[i]) && !IsSpace(b1[i]))
	}
	return isText
}

func check(file string, e error, idx int) {
	if e == io.EOF {
		return
	}
	if e != nil {
		fmt.Printf("Err idx=%v, file %s, err %s", idx, file, e)
		panic(e)
	}
}

func getAllFilesInDir(path string, allFiles []string) []string {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		var fPath = filepath.Join(path, file.Name())
		if !file.IsDir() {
			allFiles = append(allFiles, fPath)
		} else {
			allFiles = getAllFilesInDir(fPath, allFiles)
		}
	}
	return allFiles
}

func CompressFile(file string) {
	zipFile := strings.ReplaceAll(file, ".txt", ".zip")
	entry := filepath.Base(file) //pega apenas o nome do arquivo

	fmt.Printf("Zipando %s em %s, entry=%s\n", file, zipFile, entry)

	_, err := os.Stat(zipFile)
	if !os.IsNotExist(err) {
		//remove o zip arquivo caso exista
		err := os.Remove(zipFile)
		if err != nil {
			log.Fatal("Err Remove zip=", err)
		}
	}

	archive, err := os.Create(zipFile)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	f1, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	w1, err := zipWriter.Create(entry)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}
}
