// +build amd64

#include "textflag.h"

// func RgbaToBgraSIMD(data []byte)
TEXT ·RgbaToBgraSIMD(SB), NOSPLIT, $0-16
    // data pointer in RDI
    // length in RSI

    MOVQ data+0(FP), SI        // SI = &data[0]
    MOVQ data+8(FP), CX        // CX = len(data)

loop:
    CMPQ CX, $4
    JL done                    // if less then 4 - exit

    MOVB (SI), AL              // AL = R
    MOVB 2(SI), BL             // BL = B

    MOVB BL, (SI)              // data[0] = B
    MOVB AL, 2(SI)             // data[2] = R

    ADDQ $4, SI                // next pixel
    SUBQ $4, CX
    JMP loop

done:
    RET
