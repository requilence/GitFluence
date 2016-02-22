    var rs;

    function getDataByEmail(email){
     for (key in window.rs.Users) {
                  if (window.rs.Users[key].Email==email){
                  return window.rs.Users[key];
                }
               }}

(function($) {
    "use strict";


    new WOW().init();
    $(".select-repo").bind('click', function(event) {

        $("#repo").val($(this).text())
    });
    $('a.page-scroll').bind('click', function(event) {
        var $ele = $(this);
        $('html, body').stop().animate({
            scrollTop: ($($ele.attr('href')).offset().top - 60)
        }, 1450, 'easeInOutExpo');
        event.preventDefault();
    });

    $('#visbut').bind('click', function(event) {
    clearTimeout(visGetRepo);
           getRepo();
        });
        var visGetRepo;

        function attachSVG(){
        $('svg g').bind('mouseover', function(event) {
           var email=$(this).attr("id");
            if(typeof email=="undefined"){
                  $(this).tooltip({title: "Other users: "+ rs.CodeLines.Total+" lines"})
            }else{
                var data=getDataByEmail(email);
                if (typeof data!=="undefined"){
                     $(this).tooltip({title: data.Username+": " + data.CodeLines.Total+" lines"})
                }

            }
        });
        }
function getRepo(){
    var feedback = $.ajax({
        method: "POST",
        dataType: "json",
        url: "/check",
       data: { name: $("#owner").val(), repo:  $("#repo").val() }
    }) .done(function( data ) {

         if(data.status=="processing"){
           visGetRepo=setTimeout(getRepo,1000);
            $("#vis").show();
            $("#vis .loading").show();
         }else  if(data.status=="ready"){
         window.rs=data.stat;
           $("#vis #svg").show();
            $("#vis").show();
            $("#vis .loading").hide();

$.ajax({
        method: "GET",
        dataType: "text",
        url: "/static/svgs/"+data.hash+".svg",
    })  .done(function( data ) {
     $("#svg").html("");
      $("#svg").append(data);
setTimeout(attachSVG,1000);
    });

          //  $("#svg img").attr("src","/static/svgs/"+data.hash+".svg").show()




}})
}

}(jQuery));