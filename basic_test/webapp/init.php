<?php

function myPrint($arr){
	var_dump($arr);
}
function init() {
	echo "Initializing the web application";
	$res = test();
	myPrint($res);
}

function test() {
	$func = "get_defined_functions";
	$res = $func();
	return $res;
}










init();
