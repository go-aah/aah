package controllers

import (
  "aahframe.work/aah/aah.v0"

  "vfstest/app/models"
)

// AppController struct application controller
type AppController struct {
  *aah.Context
}

// Index method is application home page.
func (a *AppController) Index() {
  data := aah.Data{
    "Greet": models.Greet{
      Message: "Welcome to aah framework - Web Application",
    },
  }

  a.Reply().Ok().HTML(data)
}
