package main

import (
	"archive/zip"
	"bufio"
	b64 "encoding/base64"
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

var FILTER_EXT = "|.dll|.exe|.zip|.pdb|.db|.ico|.png|.jpg|.jpeg|.cache|.ttf|.woff2|"

var FILTER_PATH = [3]string{`\bin\`, `\obj\`, `\.git\`}

const C_PATH = "$@$@PATH@$@$"
const C_FILE = "$@$@FILE@$@$"
const C_FILE_base64 = "X#X#FILE#X#X"

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
		Ler()
	case "E":
		Escrever()
	default:
		fmt.Println("opção não reconhecida")
	}
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

	fmt.Printf("Filtrar arquivos binários e pastas específicas ? S(im) / N(ão) ?")
	reader = bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		fmt.Println(err)
	}
	filtrarArquivos := strings.ToUpper(string(char)) == "S"

	fmt.Printf("Encoda os arquivos usando base64 ? S(im) / N(ão) ?")
	reader = bufio.NewReader(os.Stdin)
	char, _, err = reader.ReadRune()
	if err != nil {
		fmt.Println(err)
	}
	encodarBase64 := strings.ToUpper(string(char)) == "S"

	fmt.Printf("Quer zipar o arquivo depois da geração ? S(im) / N(ão) ?")
	reader = bufio.NewReader(os.Stdin)
	char, _, err = reader.ReadRune()
	if err != nil {
		fmt.Println(err)
	}
	compressAfter := strings.ToUpper(string(char)) == "S"

	start := time.Now()
	start1 := time.Now()

	destFile := EscreverFile(path, filtrarArquivos, encodarBase64)

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

func EscreverFile(path string, filtrarArquivos bool, encodarBase64 bool) string {

	allFiles := make([]string, 0)
	allFiles = getAllFilesInDir(path, allFiles)

	fmt.Printf("Processando %v arquivos...\n", len(allFiles))

	qtd := 0

	if filtrarArquivos {
		fmt.Println("Filtrando os arquivos indesejados...")
		allFiles, qtd = filterFiles(allFiles, filtrarArquivos)
		fmt.Printf("Depois do filtro : %v arquivos\n", qtd)
	}
	//ordenando os arquivos por nome
	//serve apenas para poder comparar o outro arquivo gerado pela rotina .Net
	fmt.Println("Ordenando os arquivos por nome...")
	//sort.Strings(allFiles)	//esse metodo não case insensitive!
	sort.Slice(allFiles, func(i, j int) bool { return strings.ToLower(allFiles[i]) < strings.ToLower(allFiles[j]) })

	fmt.Println("Processamento dos arquivos elegíveis...")

	//destFile := `C:\temp\test.txt`
	destFile := fmt.Sprint(path, "_go.gp")

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
	if _, err := fDest.Write([]byte(fmt.Sprintf("%s=%s", C_PATH, path))); err != nil {
		fDest.Close() // ignore error; Write error takes precedence
		log.Fatal("Err Write 1=", err)
	}

	qtd = 0
	for _, f := range allFiles {

		if f == "" {
			continue
		}

		qtd++

		//header que sinaliza o conteudo de um arquivo
		var hdrFile string
		if encodarBase64 {
			hdrFile = fmt.Sprintf("\r\n%s=%s\r\n", C_FILE_base64, f)
		} else {
			hdrFile = fmt.Sprintf("\r\n%s=%s\r\n", C_FILE, f)
		}

		//grava caminho arquivo atual
		if _, err := fDest.Write([]byte(hdrFile)); err != nil {
			fDest.Close() // ignore error; Write error takes precedence
			log.Fatal("Err Write 2=", err)
		}

		//ler conteudo arquivo atual
		byt, err := os.ReadFile(f)
		if err != nil {
			log.Fatal("Err ReadFile=", err)
		}

		var s64 string
		var byt2 []byte
		if encodarBase64 {
			//encoda em base 64
			s64 = b64.StdEncoding.EncodeToString(byt)
			byt2 = make([]byte, len(s64))
			copy(byt2, []byte(s64))
		} else {
			byt2 = make([]byte, len(byt))
			copy(byt2, byt)
		}

		//grava conteudo arquivo atual
		if _, err := fDest.Write(byt2); err != nil {
			fDest.Close() // ignore error; Write error takes precedence
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

func dirExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func filterFiles(allFiles []string, filtrarArquivos bool) ([]string, int) {
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
	zipFile := strings.ReplaceAll(file, ".gp", ".zip")
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

func Ler() error {

	fmt.Printf("Informa o arquivo a ler (+ENTER):")
	reader = bufio.NewReader(os.Stdin)
	bytes, _, err := reader.ReadLine()
	if err != nil {
		fmt.Printf("err=%v", err)
		return err
	}
	arquivo := string(bytes)

	_, err = os.Stat(arquivo)
	if os.IsNotExist(err) {
		log.Fatal("O Arquivo não foi encontrado : <{file}>")
		return nil
	}

	fmt.Println("Informa o diretorio de destino (+ENTER):")
	reader = bufio.NewReader(os.Stdin)
	bytes, _, err = reader.ReadLine()
	if err != nil {
		fmt.Printf("err=%v", err)
		return err
	}
	path := string(bytes)

	if !dirExists(path) {
		panic(fmt.Sprintf("O Caminho não foi encontrado : <%s>\n", path))

		//TODO:criar o diretorio
	}

	start1 := time.Now()

	//destFile := LerFile(arquivo, path)

	elapsed1 := time.Since(start1)

	fmt.Printf("Tempo execução Escrever= %s\n", elapsed1)

	return nil
}

// func LerFile(arquivo, path string) bool {

// 	allLines := make([]string, 0)

// 	f, err := os.Open(arquivo)
// 	if err != nil {
// 		return allLines, err
// 	}
// 	defer f.Close()

// 	//abre um scan
// 	scanner := bufio.NewScanner(f)

// 	numLinha := 1
// 	var beginPath, beginArq string

// 	for scanner.Scan() {
// 		linha := scanner.Text()
// 		tipo = linha[:len(C_PATH)]

// 		if numLinha == 1 && tipo != C_PATH {
// 			log.Fatal("A primeira linha do arquivo deve conter o Path")
// 			return false
// 		}

// 		//TODOOOOOOOOOOOOOOOOOOOOO

// 		numLinha++
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return allLines, err
// 	}

// 	// var lines = File.ReadAllLines(file);

//     // var linePath = lines[0];
//     // if (!linePath.Contains(C_PATH))
//     // {
//     //     fmt.Printf("A primeira linha do arquivo deve conter o Path");
//     //     return 0;
//     // }

//     // linePath = linePath.Replace($"{C_PATH}=", string.Empty);

//     // var count = lines.Where(l => l.StartsWith(C_FILE)).Count();

//     // fmt.Printf("Path original = {linePath}");
//     // fmt.Printf("Processando {count} arquivos...");

//     // var nextLine = "";
//     // var arqFile = "";
//     // List<string> newLines = new();

//     // for (int i = 1; i < lines.Count(); i++)
//     // {
//     //     var line = lines[i];
//     //     if (i < lines.Count() - 1)
//     //     {
//     //         nextLine = lines[i + 1];
//     //     }

//     //     if (line.StartsWith(C_FILE))
//     //     {
//     //         //arquivo atual
//     //         arqFile = line.Replace($"{C_FILE}=", string.Empty);
//     //         //novo caminho
//     //         arqFile = arqFile.Replace($"{linePath}", newPath);

//     //         CriarDiretorio(arqFile);

//     //         newLines = new List<string>();
//     //         fmt.Printf("Novo arquivo detectado = {arqFile}");
//     //     }
//     //     else
//     //     {
//     //         //Console.Write(".");
//     //         newLines.Add(line);
//     //     }

//     //     if (nextLine.StartsWith(C_FILE))
//     //     {
//     //         //finaliza o arquivo
//     //         fmt.Printf(" finalizando o arquivo.");
//     //         File.WriteAllLines(arqFile, newLines);
//     //     }
//     // }
//     // if (newLines.Any())
//     // {
//     //     //finaliza o arquivo
//     //     fmt.Printf(" finalizando o arquivo.");
//     //     File.WriteAllLines(arqFile, newLines);
//     // }

//     return 0;
// }

// static void CriarDiretorio(string file)
// {
//     var fi = new FileInfo(file);
//     if (!Directory.Exists(fi.DirectoryName))
//     {
//         fmt.Printf("Criando o diretorio {fi.DirectoryName}...");
//         Directory.CreateDirectory(fi.DirectoryName);
//     }
// }
