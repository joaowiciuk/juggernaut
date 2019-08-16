/*
 ============================================================================
 Name        : read_ads1115.c
 Author      : João Wiciuk
 Description : Prints analog values using ADS1115
 ============================================================================
 */

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <linux/i2c-dev.h>
#include <sys/ioctl.h>
#include <fcntl.h>

#include "ads1115_rpi.h"

int main(void) {
	if (openI2CBus("/dev/i2c-1") == -1) {
		return EXIT_FAILURE;
	}
	setI2CSlave(0x48);
	int i;
	for (i = 0; i < 5; i++) {
		printf("%.2f\n", readVoltage(0));
	}
	return EXIT_SUCCESS;
}
