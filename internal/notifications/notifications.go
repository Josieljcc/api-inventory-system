package notifications

import "log"

func NotifyLowStock(productName string, quantity int) {
	log.Printf("Atenção: Produto %s com estoque baixo (%d unidades)", productName, quantity)
}
