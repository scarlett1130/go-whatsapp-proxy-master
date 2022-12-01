package controllers

import (
	"context"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/leadstolink/go-whatsapp-proxy/app/dto"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Controller struct {
	dbContainer *sqlstore.Container
	client      *whatsmeow.Client
	//qrChan      <-chan whatsmeow.QRChannelItem
}

var qrCode string

func NewController(db *sqlstore.Container) *Controller {
	cntrl := Controller{
		dbContainer: db,
	}

	clientLog := waLog.Stdout("Client", os.Getenv("LOG_LEVEL"), true)
	cntrl.client = whatsmeow.NewClient(cntrl.getDevice(), clientLog)
	cntrl.client.AddEventHandler(cntrl.eventHandler)

	//if cntrl.client.Store.ID == nil {
	//	qrChan, err := cntrl.client.GetQRChannel(context.Background())
	//
	//	if err != nil {
	//		cntrl.client.Log.Errorf("QR code channel error: %s", err.Error())
	//	}
	//
	//	cntrl.qrChan = qrChan
	//}

	return &cntrl
}

func (k *Controller) Login(c *fiber.Ctx) error {
	if k.client.Store.ID == nil {
		// No ID stored, new login
		if !k.client.IsConnected() {
			err := k.client.Connect()

			if err != nil {
				k.client.Log.Errorf("WhatsApp connection error: %s", err.Error())

				return c.SendStatus(500)
			}
		}

		//go func() { k.qrChan
		// client should be disconnected here
		k.client.Disconnect()
		// This must be called *before* Connect(). It will then listen to all the relevant events from the client.
		qrChan, err := k.client.GetQRChannel(context.Background())
		err = k.client.Connect()
		// connect should be after
		if err != nil {
			k.client.Log.Errorf("WhatsApp connection error: %s", err.Error())

			return c.SendStatus(500)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				//qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				qrCode = evt.Code

				if qrCode != "" {
					qrCodeImg, err := qrcode.Encode(qrCode, qrcode.Medium, 256)
					if err != nil {
						k.client.Log.Errorf("QR code generation error: %s", err.Error())

						c.SendStatus(500)
					}

					return c.Send(qrCodeImg)
				}
			} else {
				// login event that we do not catch
				//qrCode = ""
				break
			}
		}
		//}()

		//else if k.client.IsConnected() {
		//	k.client.Disconnect()
		//
		//	err := k.client.Connect()
		//	if err != nil {
		//		k.client.Log.Errorf("WhatsApp connection error: %s", err.Error())
		//		return c.SendStatus(500)
		//	}
		//}
	} else {
		// Already logged in, just connect
		if err := k.Autologin(); err != nil {
			k.client.Log.Errorf("WhatsApp connection error: %s", err.Error())

			return c.SendStatus(500)
		}

		return c.JSON(dto.Response{Status: true})
	}

	return c.JSON(dto.Response{Status: false})
}

func (k *Controller) Autologin() error {
	// autologin only when client is auth
	if k.client.Store.ID != nil && !k.client.IsConnected() {
		err := k.client.Connect()
		if err != nil {
			k.client.Log.Errorf("WhatsApp connection error: %s", err.Error())
			return err
		}
	}

	return nil
}

func (k *Controller) Logout(c *fiber.Ctx) error {
	if k.client != nil {
		return k.client.Logout()
	}

	return c.JSON(dto.Response{Status: false})
}

func (k *Controller) Shutdown(c *fiber.Ctx) error {
	os.Exit(0)

	return c.JSON(dto.Response{Status: false})
}

func (k *Controller) Restart(c *fiber.Ctx) error {
	if _, err := os.Stat("./whatsappstore.db"); err == nil {
		e := os.Remove("./whatsappstore.db")
		if e != nil {
			log.Fatal(e)
		}
	}

	cmd1 := exec.Command("./run.sh")
	// this has to be orphan process, so no need to write wait function.
	cmd1.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err1 := cmd1.Start()
	if err1 != nil {
		log.Fatal(err1)
	}
	return c.JSON(dto.Response{Status: true})
}

func (k *Controller) getDevice() *store.Device {
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := k.dbContainer.GetFirstDevice()

	if err != nil {
		k.client.Log.Errorf("Device getting error: %s", err.Error())

		return nil
	}

	return deviceStore
}

func (k *Controller) GetClient() *whatsmeow.Client {
	return k.client
}
