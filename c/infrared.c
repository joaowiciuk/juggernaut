#include "infrared.h"
#include <stdio.h>

int number() {
	return 42;
}

int send(int pin, unsigned int signal) {
    return pin;
}

int hello() {
    return 9;
}

int greet(const char *name, int year, char *out) {
    int n;
    
    n = sprintf(out, "Greetings, %s from %d! We come in peace :)", name, year);

    return n;
}