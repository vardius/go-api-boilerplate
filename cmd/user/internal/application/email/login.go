package email

const loginHTML = `
<!DOCTYPE html
  PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="en" xml:lang="en">
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta content="width=device-width, initial-scale=1" name="viewport" />
    <title>{{ .Title }}</title>
    <style type="text/css">
     @media only screen {
       html {
         min-height: 100%;
         background: #fff
       }
     }

     @media only screen and (max-width:720px) {
       .small-float-center {
         margin: 0 auto !important;
         float: none !important;
         text-align: center !important
       }
     }

     @media only screen and (max-width:696px) {
       .masthead {
         margin: 0 !important
       }
     }

     @media only screen and (max-width:696px) {
       .disclaimer {
         padding-left: 30px !important;
         padding-right: 30px !important
       }
     }
    </style>
  </head>

  <body>
    <a href={{ .LoginURL }}>Login</a>

    <!-- prevent Gmail on iOS font size manipulation -->
    <div style="display:none;white-space:nowrap;font:15px courier;line-height:0">&nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp;
      &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp;
      &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp; &nbsp;</div>
  </body>
</html>
`
