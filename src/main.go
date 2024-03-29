package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type Aula struct {
	dia        string
	inicio     time.Time
	fim        time.Time
	total      int
	disponivel int
	inscritos  int
}

func (aula Aula) toString() string {
	return fmt.Sprintf("Dia: %s\nInicio: %s\nFim: %s\nTotal: %d\nDisponivel: %d\nInscritos: %d\n\n",
		aula.dia, aula.inicio.Format("15:04"), aula.fim.Format("15:04"), aula.total, aula.disponivel, aula.inscritos)
}

func main() {
	err := configurarDB(os.Getenv("DB_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer fecharDB()

	channelID, err := strconv.ParseInt(os.Getenv("TELEGRAM_CHANNEL_ID"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	telegramChannel, err := configurarTelegram(os.Getenv("TELEGRAM_API_KEY"), channelID)
	if err != nil {
		log.Fatal(err)
	}

	atualizaAulaChannel := make(chan []Aula)
	go atualizaAulas(atualizaAulaChannel)

	for {
		select {
		case telegramUpdate := <-telegramChannel:
			switch telegramUpdate.Message.Text {
			case "/vagas":
				err = enviaAulaFiltro(telegramUpdate.Message.Chat.ID, func(aula Aula) bool {
					return aula.disponivel > 0
				})
				if err != nil {
					log.Print(err)
				}
			case "/todos":
				err = enviaAulaFiltro(telegramUpdate.Message.Chat.ID, nil)
				if err != nil {
					log.Print(err)
				}
			}
		case aulas := <-atualizaAulaChannel:
			var diferencas map[int]Aula
			diferencas, err = obterDiferencasFromDB(aulas)

			for oldMessageID, aula := range diferencas {
				var newMessageID int
				newMessageID, err = updateAulaOnChannel(oldMessageID, aula)
				if err != nil {
					log.Print(err)
					continue
				}

				err = atualizarAulaOnDB(oldMessageID, newMessageID, aula)
				if err != nil {
					log.Print(err)
				}
			}
		}
	}
}

func enviaAulaFiltro(chatID int64, filtro func(aula Aula) bool) (err error) {
	aulas, err := getAulasFromDB()
	if err != nil {
		return
	}

	for _, aula := range aulas {
		if filtro == nil || filtro(aula) {
			err = sendAulaToUser(chatID, aula)
			if err != nil {
				return
			}
		}
	}
	return
}

func atualizaAulas(atualizaAulasChannel chan<- []Aula) {
	for {
		aulas, err := getAulasFromWeb()
		if err != nil {
			log.Print(err)
		} else {
			atualizaAulasChannel <- aulas
		}

		time.Sleep(10 * time.Minute)
	}
}
